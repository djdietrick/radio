// Package auth handles password hashing (bcrypt) and stateless bearer tokens
// (JWT, HMAC-signed). Tokens carry the user id and admin flag so middleware can
// authorize requests without a database round-trip.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidToken is returned when a token fails parsing or signature checks.
var ErrInvalidToken = errors.New("auth: invalid token")

// Claims is the JWT payload for an authenticated user.
type Claims struct {
	UserID  string `json:"uid"`
	IsAdmin bool   `json:"adm"`
	jwt.RegisteredClaims
}

// Authenticator issues and verifies tokens with a fixed secret and TTL.
type Authenticator struct {
	secret []byte
	ttl    time.Duration
}

func NewAuthenticator(secret []byte, ttl time.Duration) *Authenticator {
	return &Authenticator{secret: secret, ttl: ttl}
}

// HashPassword returns a bcrypt hash suitable for storage.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword reports whether plain matches the stored bcrypt hash.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// Issue mints a signed token for the given user.
func (a *Authenticator) Issue(userID string, isAdmin bool) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(a.ttl)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(a.secret)
}

// Verify parses and validates a token string, returning its claims.
func (a *Authenticator) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method %v", ErrInvalidToken, t.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if claims.UserID == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
