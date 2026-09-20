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
	"gorm.io/gorm"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/domain/entity"
	"github.com/chun/kada-backend/internal/infra/captcha"
	"github.com/chun/kada-backend/internal/infra/sms"
	"github.com/chun/kada-backend/internal/middleware"
)

// SMSSender is the SMS sending interface
type SMSSender interface {
	SendVerificationCode(phone string) (code string, err error)
	CheckVerificationCode(phone, code string) (bool, error)
}

// Sending an SMS costs money and can be used to harass whoever owns the number, so three independent
// quotas have to hold before a message leaves the building:
//
//	per phone, 60s   - stops a single number being bombed and makes each attempt cost the attacker time;
//	per phone, 10/day - bounds the damage to one victim to ten messages a day;
//	per IP, 10/hour and 30/day - bounds a script that walks through a list of victim numbers, which the
//	                  per-phone quota alone would never notice (each number is used once).
//
// The per-IP figures are deliberately looser than the per-phone ones because a campus or office NAT puts
// many legitimate users behind one address; they are there to stop a bulk run, not to police a household.
const (
	smsPhoneCooldown = 60 * time.Second
	smsPhoneDailyMax = 10
	smsIPHourlyMax   = 10
	smsIPDailyMax    = 30

	// smsProviderThrottleWait is how long a user is told to wait when the provider itself refuses a send.
	// Aliyun's send interval defaults to 60 seconds, which is also this service's per-phone cooldown, so
	// both sides agree on the window that has to be sat out.
	smsProviderThrottleWait = 60
)

// Graphical challenge lifetime and tolerance. Five minutes is long enough to read a distorted code and
// type it, and short enough that a harvested id is useless. Five attempts bounds an automated solver.
const (
	captchaTTL         = 5 * time.Minute
	captchaMaxAttempts = 5
)

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

// normalizeEmail lower-cases the email and trims whitespace so the same canonical form is stored and
// queried. Email is no longer a sign-in method, but it is still an optional profile field.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// GenerateCaptcha issues a graphical challenge for the client to solve before it may request an SMS.
//
// Expired rows are purged here rather than by a background job: this is the only place that creates them,
// so the table cannot grow without bound and the deployment needs no scheduler.
func (s *AuthService) GenerateCaptcha(ctx context.Context, ip string) (*domain.CaptchaResponse, error) {
	code, err := captcha.Code()
	if err != nil {
		return nil, errors.New("failed to generate a verification code, please try again later")
	}

	id, err := randomID()
	if err != nil {
		return nil, errors.New("failed to generate a verification code, please try again later")
	}

	row := entity.LoginCaptcha{
		ID:        id,
		CodeHash:  sha256Hex(captcha.Normalize(code)),
		IP:        stringPtr(ip),
		ExpiresAt: time.Now().Add(captchaTTL),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		log.Printf("store captcha failed: %v", err)
		return nil, errors.New("failed to generate a verification code, please try again later")
	}

	// Housekeeping: anything that expired more than an hour ago can never be answered.
	if err := s.db.WithContext(ctx).Exec(
		`DELETE FROM login_captchas WHERE expires_at < NOW() - INTERVAL '1 hour'`).Error; err != nil {
		log.Printf("purge expired captchas failed: %v", err)
	}

	return &domain.CaptchaResponse{ID: id, Image: captcha.DataURI(code)}, nil
}

// consumeCaptcha validates an answer and burns it in a single statement.
//
// The UPDATE ... WHERE used = FALSE is the whole point: two concurrent requests carrying the same solved
// challenge cannot both succeed, so one solved captcha cannot be reused to fan out SMS sends.
func (s *AuthService) consumeCaptcha(ctx context.Context, id, answer string) error {
	if id == "" || answer == "" {
		return errors.New("the graphical verification code is required")
	}

	var consumed string
	err := s.db.WithContext(ctx).Raw(`
		UPDATE login_captchas SET used = TRUE
		WHERE id = ?
		  AND code_hash = ?
		  AND used = FALSE
		  AND expires_at > NOW()
		  AND attempts < ?
		RETURNING id
	`, id, sha256Hex(captcha.Normalize(answer)), captchaMaxAttempts).Scan(&consumed).Error
	if err == nil && consumed != "" {
		return nil
	}
	if err != nil {
		log.Printf("consume captcha %s failed: %v", id, err)
	}

	// Count the failure against every still-usable challenge for this id, so a solver cannot simply retry
	// the same challenge forever.
	_ = s.db.WithContext(ctx).Exec(`
		UPDATE login_captchas SET attempts = attempts + 1
		WHERE id = ? AND used = FALSE AND expires_at > NOW()
	`, id).Error

	return errors.New("the graphical verification code is incorrect or has expired")
}

// SendSMSCode sends an SMS verification code.
//
// The order of the checks is deliberate and is the order in which each one can reject the request most
// cheaply: the captcha (fails a script outright and costs nothing), the phone quotas, then the IP quotas,
// and only then the provider call, which is the only step that costs money.
func (s *AuthService) SendSMSCode(ctx context.Context, phone, ip, captchaID, captchaAnswer string) error {
	if !phonePattern.MatchString(phone) {
		return errors.New("invalid phone number format")
	}

	if err := s.consumeCaptcha(ctx, captchaID, captchaAnswer); err != nil {
		return err
	}

	// 60-second cooldown per phone number: prevents SMS bombing of a single number
	var recent int64
	if err := s.db.WithContext(ctx).Model(&entity.SMSVerificationCode{}).
		Where("phone = ? AND created_at > NOW() - INTERVAL '60 seconds'", phone).
		Count(&recent).Error; err == nil && recent > 0 {
		return &domain.RateLimitError{
			Message:           "too many requests, please try again in 60 seconds",
			RetryAfterSeconds: int(smsPhoneCooldown.Seconds()),
		}
	}

	// daily cap of 10 per phone number: prevents bulk bombing and SMS cost loss
	var daily int64
	if err := s.db.WithContext(ctx).Model(&entity.SMSVerificationCode{}).
		Where("phone = ? AND created_at > NOW() - INTERVAL '24 hours'", phone).
		Count(&daily).Error; err == nil && daily >= smsPhoneDailyMax {
		return &domain.RateLimitError{Message: "this phone number has reached its daily send limit, please try again tomorrow"}
	}

	// Per-IP quotas. They are skipped when the address is unknown: recording every unnamed caller as
	// 0.0.0.0 and then counting them together would lock out the whole world the moment one script ran.
	if ip != "" && ip != "0.0.0.0" {
		var fromIPHour int64
		if err := s.db.WithContext(ctx).Model(&entity.SMSVerificationCode{}).
			Where("ip = ? AND created_at > NOW() - INTERVAL '1 hour'", ip).
			Count(&fromIPHour).Error; err == nil && fromIPHour >= smsIPHourlyMax {
			return &domain.RateLimitError{
				Message:           "too many verification codes requested from this network, please try again later",
				RetryAfterSeconds: 3600,
			}
		}

		var fromIPDay int64
		if err := s.db.WithContext(ctx).Model(&entity.SMSVerificationCode{}).
			Where("ip = ? AND created_at > NOW() - INTERVAL '24 hours'", ip).
			Count(&fromIPDay).Error; err == nil && fromIPDay >= smsIPDailyMax {
			return &domain.RateLimitError{Message: "this network has reached its daily limit for verification codes, please try again tomorrow"}
		}
	}

	var code string
	var err error

	if s.sms != nil {
		code, err = s.sms.SendVerificationCode(phone)
		if err != nil {
			log.Printf("send sms code to %s failed: %v", phone, err)
			return smsSendError(err)
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
		VALUES (?, ?, ?, ?)
	`, phone, sha256Hex(code), stringPtr(ip), time.Now().Add(5*time.Minute)).Error; err != nil {
		log.Printf("store sms code failed: %v", err)
		return errors.New("failed to store verification code, please try again later")
	}

	return nil
}

// smsSendError turns a sender failure into the error the API should report.
//
// The provider's own rate limit is a wait, not a mistake, and its error code is not something a user can
// act on: answering 400 with "biz.FREQUENCY" buried in the message is how an English provider code ended up
// on a Chinese sign-in form. A throttle becomes a quota instead, which the handler already answers as a 429
// with a retry-after the client can count down.
//
// Every other failure keeps its cause in the chain. The provider code is what identifies an unapproved
// signature or template, and without it a failed sign-up is undebuggable; the user still only ever sees the
// flat "failed to send SMS" text unless the service runs outside release mode.
func smsSendError(err error) error {
	if errors.Is(err, sms.ErrProviderThrottled) {
		return &domain.RateLimitError{
			Message:           "verification codes are being requested too often, please try again in 60 seconds",
			RetryAfterSeconds: smsProviderThrottleWait,
		}
	}
	return fmt.Errorf("failed to send SMS: %w", err)
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

// LoginByEmail and RegisterByEmail used to live here. They are gone: signing in happens by phone number
// only (see entity.User), and `PATCH /api/me` remains the way an email is attached to a profile.

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

// randomID returns 32 hex characters from crypto/rand: the opaque handle for a captcha.
//
// It is not a UUID because nothing here needs the version/variant bits; what matters is that it cannot be
// guessed from another challenge, which a counter or a timestamp would allow.
func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}

// stringPtr returns nil for an empty string, so "no address" is stored as SQL NULL rather than as ”.
// An empty string would otherwise be a value a COUNT query matches, quietly lumping every unnamed caller
// into one quota bucket.
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
