package utils

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	MSISDN string `json:"msisdn"`
	jwt.RegisteredClaims
}

func GenerateToken(msisdn string, secret []byte) (string, error) {
	claims := TokenClaims{
		msisdn,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // 7-day expiry (REQ-1.2)
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
