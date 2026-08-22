package auth

import (
	"crypto/subtle"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"goscrapy/internal/timeutil"
)

const (
	DefaultUser     = "admin"
	DefaultPassword = "Admin@12345"
	Issuer          = "goscrapy"
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type Service struct {
	secret []byte
	expire time.Duration
}

func New(secret string, expire time.Duration) *Service {
	if expire <= 0 {
		expire = 24 * time.Hour
	}
	if secret == "" {
		secret = "goscrapy-dev-secret"
	}
	return &Service{secret: []byte(secret), expire: expire}
}

func (s *Service) Authenticate(username, password string) (token string, expiresIn int64, err error) {
	if !match(username, DefaultUser) || !match(password, DefaultPassword) {
		return "", 0, ErrBadCredential
	}
	now := timeutil.Now()
	expiresIn = int64(s.expire.Seconds())
	claims := Claims{
		Username: DefaultUser,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   DefaultUser,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expire)),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Second)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, expiresIn, nil
}

func (s *Service) Parse(token string) (*Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrUnauthorized
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, ErrUnauthorized
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || claims.Username == "" {
		return nil, ErrUnauthorized
	}
	return claims, nil
}

func match(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

var (
	ErrBadCredential = fmt.Errorf("bad credential")
	ErrUnauthorized  = fmt.Errorf("unauthorized")
)
