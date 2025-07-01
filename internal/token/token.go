package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	jwtSecret []byte
)

func init() {
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
}

type Claims struct {
	UserGUID string `json:"user_guid"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userGUID string, expireMinutes int) (string, error) {
	expirationTime := time.Now().Add(time.Duration(expireMinutes) * time.Minute)
	claims := &Claims{
		UserGUID: userGUID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString(jwtSecret)
}

func ValidateAccessToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserGUID, nil
	}

	return "", errors.New("invalid token")
}

func GenerateRefreshToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func ValidateRefreshToken(token string, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(token))
}
