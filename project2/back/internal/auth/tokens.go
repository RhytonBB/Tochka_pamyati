package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}

func SignHS256(secret []byte, userID uuid.UUID, tokenType string, exp time.Time, jti uuid.UUID) (string, error) {
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        jti.String(),
		},
		Type: tokenType,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

func ParseHS256(secret []byte, token, expectedType string) (TokenClaims, error) {
	var claims TokenClaims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil || !parsed.Valid {
		return TokenClaims{}, errors.New("invalid token")
	}
	if claims.Type != expectedType {
		return TokenClaims{}, errors.New("invalid token type")
	}
	return claims, nil
}
