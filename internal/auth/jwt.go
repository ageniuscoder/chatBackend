package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserId int64 `json:"userid"`
	jwt.RegisteredClaims
}

func NewToken(secret string, userid int64, ttlmin int) (string, error) {
	claims := Claims{
		UserId: userid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(ttlmin) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "mmchat",
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func ParseToken(secret, token string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(token, Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := tok.Claims.(*Claims); ok && tok.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
