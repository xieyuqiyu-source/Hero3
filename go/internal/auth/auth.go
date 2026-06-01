// Package auth 提供 JWT 认证 + 玩家归属校验
package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ctxAccountID contextKey = "accountId"
	ctxIsAdmin   contextKey = "isAdmin"
)

var (
	ErrMissingToken    = errors.New("missing token")
	ErrInvalidToken    = errors.New("invalid token")
	ErrExpiredToken    = errors.New("expired token")
	ErrPermissionDenied = errors.New("permission denied")
)

// Config 认证配置
type Config struct {
	JWTSecret  string        // JWT 签名密钥
	AdminToken string        // Admin 静态 token（用于 GM 后台）
	TokenTTL   time.Duration // JWT 有效期
}

// Claims JWT 载荷
type Claims struct {
	AccountID string `json:"accountId"`
	jwt.RegisteredClaims
}

// IssueToken 签发 JWT（登录时调用）
func IssueToken(cfg Config, accountID string) (string, error) {
	if cfg.JWTSecret == "" {
		return "", errors.New("jwt secret not configured")
	}

	ttl := cfg.TokenTTL
	if ttl == 0 {
		ttl = 7 * 24 * time.Hour // 默认 7 天
	}

	claims := Claims{
		AccountID: accountID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// ParseToken 解析 JWT，返回 accountID
func ParseToken(cfg Config, tokenString string) (string, error) {
	if cfg.JWTSecret == "" {
		return "", errors.New("jwt secret not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(cfg.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}

	return claims.AccountID, nil
}

// AccountIDFromContext 从 context 中取出 accountID（中间件已验证过）
func AccountIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxAccountID).(string)
	return id, ok && id != ""
}

// IsAdminFromContext 从 context 中判断是否是 admin 请求
func IsAdminFromContext(ctx context.Context) bool {
	v, ok := ctx.Value(ctxIsAdmin).(bool)
	return ok && v
}

// AuthMiddleware 玩家认证中间件
// - 从 Authorization: Bearer <token> 解析 JWT
// - 把 accountID 放进 context
// - 公开路由（whitelist 中）跳过认证
func AuthMiddleware(cfg Config, publicPaths []string) func(http.Handler) http.Handler {
	publicSet := make(map[string]bool, len(publicPaths))
	for _, p := range publicPaths {
		publicSet[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Admin token 优先（用于 GM 后台）
			if adminToken := r.Header.Get("X-Admin-Token"); adminToken != "" {
				if cfg.AdminToken != "" && adminToken == cfg.AdminToken {
					ctx := context.WithValue(r.Context(), ctxIsAdmin, true)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// 提供了 admin token 但不匹配 → 拒绝
				http.Error(w, "invalid admin token", http.StatusUnauthorized)
				return
			}

			// 2. 公开路径直接放行（但如果有 token 也尝试解析，便于后续 handler 做归属校验）
			path := r.URL.Path
			if isPublicPath(path, publicSet) {
				if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
					tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
					if accountID, err := ParseToken(cfg, tokenStr); err == nil {
						ctx := context.WithValue(r.Context(), ctxAccountID, accountID)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
				next.ServeHTTP(w, r)
				return
			}

			// 3. Admin 路径必须用 admin token，没传就拒绝
			if strings.HasPrefix(path, "/api/v1/admin/") {
				http.Error(w, "admin access required", http.StatusForbidden)
				return
			}

			// 4. 其他路径需要 JWT
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			accountID, err := ParseToken(cfg, tokenStr)
			if err != nil {
				http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxAccountID, accountID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// isPublicPath 判断路径是否是公开的（不需要认证）
func isPublicPath(path string, publicSet map[string]bool) bool {
	if publicSet[path] {
		return true
	}
	// 战报详情公开（用于分享）
	if strings.HasPrefix(path, "/api/v1/reports/") {
		return true
	}
	return false
}
