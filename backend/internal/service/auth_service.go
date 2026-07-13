package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/middleware"
)

// SMSSender 短信发送接口
type SMSSender interface {
	SendVerificationCode(phone string) (code string, err error)
	CheckVerificationCode(phone, code string) (bool, error)
}

type AuthService struct {
	db        *pgxpool.Pool
	jwtSecret string
	jwtExpire time.Duration
	sms       SMSSender // 短信发送器
}

func NewAuthService(db *pgxpool.Pool, jwtSecret, jwtExpire string, sms SMSSender) *AuthService {
	d, _ := time.ParseDuration(jwtExpire)
	return &AuthService{db: db, jwtSecret: jwtSecret, jwtExpire: d, sms: sms}
}

// SendSMSCode 发送短信验证码
func (s *AuthService) SendSMSCode(ctx context.Context, phone string) error {
	if len(phone) != 11 {
		return errors.New("手机号格式不正确")
	}

	var code string
	var err error

	if s.sms != nil {
		code, err = s.sms.SendVerificationCode(phone)
		if err != nil {
			return fmt.Errorf("短信发送失败: %w", err)
		}
	} else {
		code = generateSMSCode()
		fmt.Printf("📱 [DEV] Phone: %s, Code: %s\n", phone, code)
	}

	// 存储验证码到数据库（5分钟有效）
	_, err = s.db.Exec(ctx, `
		INSERT INTO sms_codes (phone, code, ip, expires_at)
		VALUES ($1, $2, '0.0.0.0', $3)
	`, phone, code, time.Now().Add(5*time.Minute))
	if err != nil {
		return fmt.Errorf("存储验证码失败: %w", err)
	}

	return nil
}

// LoginByPhone 手机号+验证码登录
	var err error
func (s *AuthService) LoginByPhone(ctx context.Context, phone, code string) (*domain.AuthResponse, error) {
	// 查数据库校验验证码
	var count int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM sms_codes
		WHERE phone = $1 AND code = $2 AND used = FALSE AND expires_at > NOW()
	`, phone, code).Scan(&count); err != nil || count == 0 {
		return nil, errors.New("验证码错误或已过期")
	}

	// 标记验证码已使用
	_, _ = s.db.Exec(ctx, `UPDATE sms_codes SET used = TRUE WHERE phone = $1 AND code = $2`, phone, code)

	// 查找或创建用户
	var user domain.UserInfo
	err = s.db.QueryRow(ctx, `
		SELECT id, phone, email, name, avatar FROM users WHERE phone = $1
	`, phone).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)

	if err != nil {
		// 新用户，自动注册
		err = s.db.QueryRow(ctx, `
			INSERT INTO users (phone) VALUES ($1)
			RETURNING id, phone, email, name, avatar
		`, phone).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)
		if err != nil {
			return nil, fmt.Errorf("创建用户失败: %w", err)
		}
	}

	// 更新最后登录
	_, _ = s.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	// 生成 JWT
	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: user}, nil
}

// LoginByEmail 邮箱+密码登录
func (s *AuthService) LoginByEmail(ctx context.Context, email, password string) (*domain.AuthResponse, error) {
	var user domain.UserInfo
	var passwordHash string
	var err error

	err = s.db.QueryRow(ctx, `
		SELECT id, phone, email, name, avatar, COALESCE(password_hash, '')
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar, &passwordHash)
	if err != nil {
		return nil, errors.New("邮箱或密码错误")
	}

	if passwordHash == "" {
		return nil, errors.New("该账号未设置密码，请使用手机号登录")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, errors.New("邮箱或密码错误")
	}

	_, _ = s.db.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: user}, nil
}

// RegisterByEmail 邮箱注册
func (s *AuthService) RegisterByEmail(ctx context.Context, email, password, name string) (*domain.AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	var user domain.UserInfo
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3)
		RETURNING id, phone, email, name, avatar
	`, email, string(hash), name).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)
	if err != nil {
		return nil, fmt.Errorf("注册失败，邮箱可能已被使用: %w", err)
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{Token: token, User: user}, nil
}

// GetUserByID 获取用户信息
func (s *AuthService) GetUserByID(ctx context.Context, userID int64) (*domain.UserInfo, error) {
	var err error
	var user domain.UserInfo
	err = s.db.QueryRow(ctx, `
		SELECT id, phone, email, name, avatar FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	return &user, nil
}

// generateToken 生成 JWT
func (s *AuthService) generateToken(user domain.UserInfo) (string, error) {
	claims := middleware.Claims{
		UserID:   user.ID,
		Phone:    user.Phone,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "kada",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// generateSMSCode 生成6位数字验证码
func generateSMSCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}
