package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/middleware"
)

// SMSSender is the SMS sending interface
type SMSSender interface {
	SendVerificationCode(phone string) (code string, err error)
	CheckVerificationCode(phone, code string) (bool, error)
}

type AuthService struct {
	db        *pgxpool.Pool
	jwtSecret string
	jwtExpire time.Duration
	sms       SMSSender // SMS sender
}

func NewAuthService(db *pgxpool.Pool, jwtSecret, jwtExpire string, sms SMSSender) *AuthService {
	d, _ := time.ParseDuration(jwtExpire)
	return &AuthService{db: db, jwtSecret: jwtSecret, jwtExpire: d, sms: sms}
}

// phonePattern matches mainland China mobile numbers: leading 1 + 3-9 + 9 digits
var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// normalizeEmail lower-cases the email and trims whitespace:
// it guarantees the same canonical form is stored and queried on registration/login/update, making lookups case-insensitive together with the LOWER(email) unique index.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// dummyPasswordHash makes the service still run one bcrypt comparison when the account does not exist or has no password set,
// evening out the response time so an attacker cannot enumerate registered emails by latency.
var dummyPasswordHash = func() string {
	h, _ := bcrypt.GenerateFromPassword([]byte("kada-timing-equalizer"), bcrypt.DefaultCost)
	return string(h)
}()

// SendSMSCode sends an SMS verification code
func (s *AuthService) SendSMSCode(ctx context.Context, phone string) error {
	if !phonePattern.MatchString(phone) {
		return errors.New("invalid phone number format")
	}

	// 60-second cooldown per phone number: prevents SMS bombing of a single number
	var recent int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM sms_codes
		WHERE phone = $1 AND created_at > NOW() - INTERVAL '60 seconds'
	`, phone).Scan(&recent); err == nil && recent > 0 {
		return errors.New("too many requests, please try again in 60 seconds")
	}

	// daily cap of 10 per phone number: prevents bulk bombing and SMS cost loss
	var daily int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM sms_codes
		WHERE phone = $1 AND created_at > NOW() - INTERVAL '24 hours'
	`, phone).Scan(&daily); err == nil && daily >= 10 {
		return errors.New("this phone number has reached its daily send limit, please try again tomorrow")
	}

	var code string
	var err error

	if s.sms != nil {
		code, err = s.sms.SendVerificationCode(phone)
		if err != nil {
			log.Printf("send sms code to %s failed: %v", phone, err)
			return errors.New("failed to send SMS, please try again later")
		}
	} else {
		code = generateSMSCode()
		// security: the plaintext code is only printed outside release mode; production logs never contain the code
		if os.Getenv("GIN_MODE") != "release" {
			fmt.Printf("📱 [DEV] Phone: %s, Code: %s\n", phone, code)
		}
	}

	// store the code hash in the database (valid for 5 minutes): no plaintext is persisted, so a database leak cannot be replayed directly
	_, err = s.db.Exec(ctx, `
		INSERT INTO sms_codes (phone, code_hash, ip, expires_at)
		VALUES ($1, $2, '0.0.0.0', $3)
	`, phone, sha256Hex(code), time.Now().Add(5*time.Minute))
	if err != nil {
		log.Printf("store sms code failed: %v", err)
		return errors.New("failed to store verification code, please try again later")
	}

	return nil
}

// LoginByPhone logs in with phone number + verification code
func (s *AuthService) LoginByPhone(ctx context.Context, phone, code string) (*domain.AuthResponse, error) {
	if !phonePattern.MatchString(phone) {
		return nil, errors.New("invalid phone number format")
	}

	// atomically consume the code: a single UPDATE validates "unused, unexpired, attempts under the limit" at once,
	// eliminating the race condition where concurrent requests consume the same code twice
	codeHash := sha256Hex(code)
	var codeID int64
	err := s.db.QueryRow(ctx, `
		UPDATE sms_codes SET used = TRUE
		WHERE id = (
			SELECT id FROM sms_codes
			WHERE phone = $1 AND code_hash = $2 AND used = FALSE
			  AND expires_at > NOW() AND attempts < 5
			ORDER BY id
			LIMIT 1
		)
		RETURNING id
	`, phone, codeHash).Scan(&codeID)
	if err != nil {
		// failed-attempt counting: increments every currently unused, unexpired pending code for the phone number.
		// previously the row was located by code, so a wrong code never matched a row and attempts was useless;
		// after switching to counting per phone number, 5 consecutive failures invalidate that phone's pending codes.
		_, _ = s.db.Exec(ctx, `
			UPDATE sms_codes SET attempts = attempts + 1
			WHERE phone = $1 AND used = FALSE AND expires_at > NOW()
		`, phone)
		return nil, errors.New("invalid or expired verification code")
	}

	// find or create the user
	var user domain.UserInfo
	err = s.db.QueryRow(ctx, `
		SELECT id, phone, email, name, avatar FROM users WHERE phone = $1
	`, phone).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)

	if err != nil {
		// new user, register automatically
		err = s.db.QueryRow(ctx, `
			INSERT INTO users (phone) VALUES ($1)
			RETURNING id, phone, email, name, avatar
		`, phone).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)
		if err != nil {
			log.Printf("create user by phone %s failed: %v", phone, err)
			return nil, errors.New("login failed, please try again later")
		}
	}

	// update last login
	_, _ = s.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	// generate JWT
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: user}, nil
}

// LoginByEmail logs in with email + password
func (s *AuthService) LoginByEmail(ctx context.Context, email, password string) (*domain.AuthResponse, error) {
	email = normalizeEmail(email)

	var user domain.UserInfo
	var passwordHash string
	var err error

	err = s.db.QueryRow(ctx, `
		SELECT id, phone, email, name, avatar, COALESCE(password_hash, '')
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar, &passwordHash)
	if err != nil {
		// account does not exist: still run one bcrypt comparison so the response time matches a wrong password, preventing email enumeration
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
		return nil, errors.New("invalid email or password")
	}

	if passwordHash == "" {
		// no password set: compare once as well and return the same error text, avoiding email enumeration and a timing side channel
		log.Printf("login attempt for user without password set: id=%d", user.ID)
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
		return nil, errors.New("invalid email or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	_, _ = s.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: user}, nil
}

// RegisterByEmail registers with email
func (s *AuthService) RegisterByEmail(ctx context.Context, email, password, name string) (*domain.AuthResponse, error) {
	email = normalizeEmail(email)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt hash failed: %v", err)
		return nil, errors.New("registration failed, please try again later")
	}

	var user domain.UserInfo
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3)
		RETURNING id, phone, email, name, avatar
	`, email, string(hash), name).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)
	if err != nil {
		log.Printf("register by email failed: %v", err)
		return nil, errors.New("registration failed, the email may already be in use")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: user}, nil
}

// GetUserByID gets user info
func (s *AuthService) GetUserByID(ctx context.Context, userID int64) (*domain.UserInfo, error) {
	var err error
	var user domain.UserInfo
	err = s.db.QueryRow(ctx, `
		SELECT id, phone, email, name, avatar FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

// UpdateUser updates user info
func (s *AuthService) UpdateUser(ctx context.Context, userID int64, name *string, email *string) (*domain.UserInfo, error) {
	// only handle non-empty fields
	var newName, newEmail *string
	if name != nil && *name != "" {
		newName = name
	}
	if email != nil && *email != "" {
		norm := normalizeEmail(*email)
		newEmail = &norm
	}

	// return the current user when there is nothing to update
	// (previously, when both fields were empty strings, the code neither ran the UPDATE nor took this branch and returned a zero-value struct)
	if newName == nil && newEmail == nil {
		return s.GetUserByID(ctx, userID)
	}

	var user domain.UserInfo
	err := s.db.QueryRow(ctx, `
		UPDATE users SET
			name = COALESCE($1, name),
			email = COALESCE($2, email),
			updated_at = NOW()
		WHERE id = $3
		RETURNING id, phone, email, name, avatar
	`, newName, newEmail, userID).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)
	if err != nil {
		log.Printf("update user %d failed: %v", userID, err)
		return nil, errors.New("failed to update user info")
	}

	return &user, nil
}

// generateToken generates a JWT
func (s *AuthService) generateToken(user domain.UserInfo) (string, error) {
	claims := middleware.Claims{
		UserID: user.ID,
		Phone:  user.Phone,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "kada",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// generateSMSCode generates a 6-digit numeric verification code
func generateSMSCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}
