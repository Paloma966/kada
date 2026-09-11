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
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/domain/entity"
	"github.com/chun/kada-backend/internal/middleware"
)

// SMSSender is the SMS sending interface
type SMSSender interface {
	SendVerificationCode(phone string) (code string, err error)
	CheckVerificationCode(phone, code string) (bool, error)
}

type AuthService struct {
	db        *gorm.DB
	jwtSecret string
	jwtExpire time.Duration
	sms       SMSSender // SMS sender
}

func NewAuthService(db *gorm.DB, jwtSecret, jwtExpire string, sms SMSSender) *AuthService {
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
	var recent int64
	if err := s.db.WithContext(ctx).Model(&entity.SMSVerificationCode{}).
		Where("phone = ? AND created_at > NOW() - INTERVAL '60 seconds'", phone).
		Count(&recent).Error; err == nil && recent > 0 {
		return errors.New("too many requests, please try again in 60 seconds")
	}

	// daily cap of 10 per phone number: prevents bulk bombing and SMS cost loss
	var daily int64
	if err := s.db.WithContext(ctx).Model(&entity.SMSVerificationCode{}).
		Where("phone = ? AND created_at > NOW() - INTERVAL '24 hours'", phone).
		Count(&daily).Error; err == nil && daily >= 10 {
		return errors.New("this phone number has reached its daily send limit, please try again tomorrow")
	}

	var code string
	var err error

	if s.sms != nil {
		code, err = s.sms.SendVerificationCode(phone)
		if err != nil {
			log.Printf("send sms code to %s failed: %v", phone, err)
			// Keep the sender's cause in the error chain instead of flattening every failure into one
			// opaque message: the provider code is what identifies an unapproved signature/template or a
			// disabled AccessKey, and without it a failed sign-up is undebuggable.
			return fmt.Errorf("failed to send SMS: %w", err)
		}
	} else {
		code = generateSMSCode()
		// security: the plaintext code is only printed outside release mode; production logs never contain the code
		if os.Getenv("GIN_MODE") != "release" {
			fmt.Printf("📱 [DEV] Phone: %s, Code: %s\n", phone, code)
		}
	}

	// store the code hash in the database (valid for 5 minutes): no plaintext is persisted, so a database leak cannot be replayed directly
	if err := s.db.WithContext(ctx).Exec(`
		INSERT INTO sms_codes (phone, code_hash, ip, expires_at)
		VALUES (?, ?, '0.0.0.0', ?)
	`, phone, sha256Hex(code), time.Now().Add(5*time.Minute)).Error; err != nil {
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
	err := s.db.WithContext(ctx).Raw(`
		UPDATE sms_codes SET used = TRUE
		WHERE id = (
			SELECT id FROM sms_codes
			WHERE phone = ? AND code_hash = ? AND used = FALSE
			  AND expires_at > NOW() AND attempts < 5
			ORDER BY id
			LIMIT 1
		)
		RETURNING id
	`, phone, codeHash).Scan(&codeID).Error
	if err != nil || codeID == 0 {
		// failed-attempt counting: increments every currently unused, unexpired pending code for the phone number.
		// previously the row was located by code, so a wrong code never matched a row and attempts was useless;
		// after switching to counting per phone number, 5 consecutive failures invalidate that phone's pending codes.
		_ = s.db.WithContext(ctx).Exec(`
			UPDATE sms_codes SET attempts = attempts + 1
			WHERE phone = ? AND used = FALSE AND expires_at > NOW()
		`, phone).Error
		return nil, errors.New("invalid or expired verification code")
	}

	// find or create the user
	user, userErr := s.userByPhone(ctx, phone)
	if userErr != nil {
		if !errors.Is(userErr, gorm.ErrRecordNotFound) {
			// A read error that is not "no such row" (connection loss, ...) must not be mistaken for a new
			// account, otherwise the INSERT below would fail with a confusing duplicate-key error.
			log.Printf("lookup user by phone %s failed: %v", phone, userErr)
			return nil, errors.New("login failed, please try again later")
		}
		// new user, register automatically
		row := entity.User{Phone: &phone}
		if createErr := s.db.WithContext(ctx).Create(&row).Error; createErr != nil {
			log.Printf("create user by phone %s failed: %v", phone, createErr)
			return nil, errors.New("login failed, please try again later")
		}
		user = toUserInfo(row)
	}

	// update last login
	_ = s.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", user.ID).
		Update("last_login_at", time.Now()).Error

	// generate JWT
	token, err := s.generateToken(*user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: *user}, nil
}

// LoginByEmail logs in with email + password
func (s *AuthService) LoginByEmail(ctx context.Context, email, password string) (*domain.AuthResponse, error) {
	email = normalizeEmail(email)

	var row entity.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&row).Error
	if err != nil {
		// account does not exist: still run one bcrypt comparison so the response time matches a wrong password, preventing email enumeration
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
		return nil, errors.New("invalid email or password")
	}

	passwordHash := ""
	if row.PasswordHash != nil {
		passwordHash = *row.PasswordHash
	}
	if passwordHash == "" {
		// no password set: compare once as well and return the same error text, avoiding email enumeration and a timing side channel
		log.Printf("login attempt for user without password set: id=%d", row.ID)
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
		return nil, errors.New("invalid email or password")
	}

	if compareErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); compareErr != nil {
		return nil, errors.New("invalid email or password")
	}

	_ = s.db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ?", row.ID).
		Update("last_login_at", time.Now()).Error

	user := toUserInfo(row)
	token, err := s.generateToken(*user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: *user}, nil
}

// RegisterByEmail registers with email
func (s *AuthService) RegisterByEmail(ctx context.Context, email, password, name string) (*domain.AuthResponse, error) {
	email = normalizeEmail(email)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt hash failed: %v", err)
		return nil, errors.New("registration failed, please try again later")
	}

	hashed := string(hash)
	row := entity.User{Email: &email, PasswordHash: &hashed, Name: &name}
	if createErr := s.db.WithContext(ctx).Create(&row).Error; createErr != nil {
		log.Printf("register by email failed: %v", createErr)
		// A duplicate email is a client mistake (409), anything else is a real server-side failure and
		// must not be disguised as one; the handler maps the sentinel to the right status.
		if isDuplicateKey(createErr) {
			return nil, domain.ErrEmailTaken
		}
		return nil, fmt.Errorf("registration failed: %w", createErr)
	}

	user := toUserInfo(row)
	token, err := s.generateToken(*user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: *user}, nil
}

// GetUserByID gets user info
func (s *AuthService) GetUserByID(ctx context.Context, userID int64) (*domain.UserInfo, error) {
	var row entity.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&row).Error; err != nil {
		return nil, errors.New("user not found")
	}
	return toUserInfo(row), nil
}

// UpdateUser updates user info
func (s *AuthService) UpdateUser(ctx context.Context, userID int64, name *string, email *string) (*domain.UserInfo, error) {
	// only handle non-empty fields
	updates := map[string]any{"updated_at": gorm.Expr("NOW()")}
	if name != nil && *name != "" {
		updates["name"] = *name
	}
	if email != nil && *email != "" {
		updates["email"] = normalizeEmail(*email)
	}

	// return the current user when there is nothing to update
	// (previously, when both fields were empty strings, the code neither ran the UPDATE nor took this branch and returned a zero-value struct)
	if len(updates) == 1 {
		return s.GetUserByID(ctx, userID)
	}

	var row entity.User
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&entity.User{}).Where("id = ?", userID).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("id = ?", userID).First(&row).Error
	}); err != nil {
		log.Printf("update user %d failed: %v", userID, err)
		return nil, errors.New("failed to update user info")
	}

	return toUserInfo(row), nil
}

// userByPhone returns the account for a phone number.
// The caller distinguishes "no such user" (gorm.ErrRecordNotFound -> sign up) from a real read error.
func (s *AuthService) userByPhone(ctx context.Context, phone string) (*domain.UserInfo, error) {
	var row entity.User
	if err := s.db.WithContext(ctx).Where("phone = ?", phone).First(&row).Error; err != nil {
		return nil, err
	}
	user := toUserInfo(row)
	return user, nil
}

// toUserInfo maps the persistence model onto the API model.
// UserInfo only exposes phone/email/name/avatar, so the password hash can never leak through it.
func toUserInfo(row entity.User) *domain.UserInfo {
	return &domain.UserInfo{
		ID:           row.ID,
		Phone:        row.Phone,
		Email:        row.Email,
		Name:         row.Name,
		Avatar:       row.Avatar,
		WechatOpenID: row.WechatOpenID,
	}
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
