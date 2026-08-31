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

// phonePattern 中国大陆手机号：1 开头 + 3-9 + 9 位数字
var phonePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// normalizeEmail 统一邮箱小写并去空白：
// 保证注册/登录/更新时以同一规范存储与查询，配合 LOWER(email) 唯一索引实现大小写不敏感。
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// dummyPasswordHash 用于账号不存在/未设密码时也做一次 bcrypt 比较，
// 抹平响应时间差，避免攻击者根据耗时枚举已注册邮箱。
var dummyPasswordHash = func() string {
	h, _ := bcrypt.GenerateFromPassword([]byte("kada-timing-equalizer"), bcrypt.DefaultCost)
	return string(h)
}()

// SendSMSCode 发送短信验证码
func (s *AuthService) SendSMSCode(ctx context.Context, phone string) error {
	if !phonePattern.MatchString(phone) {
		return errors.New("手机号格式不正确")
	}

	// 每手机号 60 秒冷却：防止对单一手机号短信轰炸
	var recent int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM sms_codes
		WHERE phone = $1 AND created_at > NOW() - INTERVAL '60 seconds'
	`, phone).Scan(&recent); err == nil && recent > 0 {
		return errors.New("发送过于频繁，请 60 秒后再试")
	}

	// 每手机号每日上限 10 条：防止批量轰炸造成短信费用损失
	var daily int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM sms_codes
		WHERE phone = $1 AND created_at > NOW() - INTERVAL '24 hours'
	`, phone).Scan(&daily); err == nil && daily >= 10 {
		return errors.New("该手机号今日发送次数已达上限，请明天再试")
	}

	var code string
	var err error

	if s.sms != nil {
		code, err = s.sms.SendVerificationCode(phone)
		if err != nil {
			log.Printf("send sms code to %s failed: %v", phone, err)
			return errors.New("短信发送失败，请稍后再试")
		}
	} else {
		code = generateSMSCode()
		// 安全：验证码明文只允许在非 release 模式下打印，生产日志绝不落验证码
		if os.Getenv("GIN_MODE") != "release" {
			fmt.Printf("📱 [DEV] Phone: %s, Code: %s\n", phone, code)
		}
	}

	// 存储验证码到数据库（5分钟有效）
	_, err = s.db.Exec(ctx, `
		INSERT INTO sms_codes (phone, code, ip, expires_at)
		VALUES ($1, $2, '0.0.0.0', $3)
	`, phone, code, time.Now().Add(5*time.Minute))
	if err != nil {
		log.Printf("store sms code failed: %v", err)
		return errors.New("验证码存储失败，请稍后再试")
	}

	return nil
}

// LoginByPhone 手机号+验证码登录
func (s *AuthService) LoginByPhone(ctx context.Context, phone, code string) (*domain.AuthResponse, error) {
	if !phonePattern.MatchString(phone) {
		return nil, errors.New("手机号格式不正确")
	}

	// 原子消耗验证码：单条 UPDATE 同时完成「未使用、未过期、尝试次数未超限」校验，
	// 消除并发请求双用同一验证码的竞态
	var codeID int64
	err := s.db.QueryRow(ctx, `
		UPDATE sms_codes SET used = TRUE
		WHERE id = (
			SELECT id FROM sms_codes
			WHERE phone = $1 AND code = $2 AND used = FALSE
			  AND expires_at > NOW() AND attempts < 5
			ORDER BY id
			LIMIT 1
		)
		RETURNING id
	`, phone, code).Scan(&codeID)
	if err != nil {
		// 失败尝试计数（验证码仍存在且未使用时），5 次后该验证码作废
		_, _ = s.db.Exec(ctx, `
			UPDATE sms_codes SET attempts = attempts + 1
			WHERE phone = $1 AND code = $2 AND used = FALSE AND expires_at > NOW()
		`, phone, code)
		return nil, errors.New("验证码错误或已过期")
	}

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
			log.Printf("create user by phone %s failed: %v", phone, err)
			return nil, errors.New("登录失败，请稍后再试")
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
	email = normalizeEmail(email)

	var user domain.UserInfo
	var passwordHash string
	var err error

	err = s.db.QueryRow(ctx, `
		SELECT id, phone, email, name, avatar, COALESCE(password_hash, '')
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar, &passwordHash)
	if err != nil {
		// 账号不存在：同样执行一次 bcrypt 比较，保持响应耗时与密码错误一致，防邮箱枚举
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
		return nil, errors.New("邮箱或密码错误")
	}

	if passwordHash == "" {
		// 未设置密码：同样比较一次并返回相同错误文案，避免邮箱枚举与时间侧信道
		log.Printf("login attempt for user without password set: id=%d", user.ID)
		_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
		return nil, errors.New("邮箱或密码错误")
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
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
	email = normalizeEmail(email)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt hash failed: %v", err)
		return nil, errors.New("注册失败，请稍后再试")
	}

	var user domain.UserInfo
	err = s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3)
		RETURNING id, phone, email, name, avatar
	`, email, string(hash), name).Scan(&user.ID, &user.Phone, &user.Email, &user.Name, &user.Avatar)
	if err != nil {
		log.Printf("register by email failed: %v", err)
		return nil, errors.New("注册失败，邮箱可能已被使用")
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

// UpdateUser 更新用户信息
func (s *AuthService) UpdateUser(ctx context.Context, userID int64, name *string, email *string) (*domain.UserInfo, error) {
	// 仅处理非空字段
	var newName, newEmail *string
	if name != nil && *name != "" {
		newName = name
	}
	if email != nil && *email != "" {
		norm := normalizeEmail(*email)
		newEmail = &norm
	}

	// 没有实际可更新的内容时返回当前用户
	// （此前实现两个字段均为空串时既不执行 UPDATE 也不走此分支，返回零值结构体）
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
		return nil, errors.New("更新用户信息失败")
	}

	return &user, nil
}

// generateToken 生成 JWT
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

// generateSMSCode 生成6位数字验证码
func generateSMSCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return fmt.Sprintf("%06d", n.Int64())
}
