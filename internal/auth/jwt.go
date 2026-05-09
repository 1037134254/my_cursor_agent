package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"my_cursor/internal/tenant"
)

type accessClaims struct {
	UserID   string   `json:"uid"`
	TenantID string   `json:"tid"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// IssueAccessJWT 签发访问令牌。
func IssueAccessJWT(rec UserRecord) (string, error) {
	sec := jwtSecret()
	if len(sec) < 32 {
		return "", errors.New("JWT_SECRET 长度至少 32 字符")
	}
	tid := tenant.SanitizeID(rec.TenantID)
	if tid == "" {
		tid = tenant.DefaultID
	}
	now := time.Now()
	claims := accessClaims{
		UserID:   rec.Username,
		TenantID: tid,
		Roles:    rec.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   rec.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL())),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(sec))
}

// ParseAccessJWT 解析访问令牌为 Principal。
func ParseAccessJWT(tokenStr string) (Principal, error) {
	sec := jwtSecret()
	if sec == "" {
		return Principal{}, errors.New("JWT_SECRET 未配置")
	}
	var claims accessClaims
	_, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected alg %v", t.Header["alg"])
		}
		return []byte(sec), nil
	})
	if err != nil {
		return Principal{}, err
	}
	tid := tenant.SanitizeID(claims.TenantID)
	if tid == "" {
		tid = tenant.DefaultID
	}
	return Principal{
		UserID:   claims.UserID,
		TenantID: tid,
		Roles:    claims.Roles,
	}, nil
}
