package jwt

import (
	"fmt"
	"time"

	"just-vpn/pkg/setting"

	golangjwt "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserId   int    `json:"user_id"`
	DeviceNo string `json:"device_no"`
	golangjwt.RegisteredClaims
}

func Generate(userId int, deviceNo string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserId:   userId,
		DeviceNo: deviceNo,
		RegisteredClaims: golangjwt.RegisteredClaims{
			ExpiresAt: golangjwt.NewNumericDate(now.Add(setting.AppConfig.JwtExpire)),
			IssuedAt:  golangjwt.NewNumericDate(now),
		},
	}
	token := golangjwt.NewWithClaims(golangjwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(setting.AppConfig.JwtSecret))
}

func Parse(tokenString string) (*Claims, error) {
	token, err := golangjwt.ParseWithClaims(tokenString, &Claims{}, func(token *golangjwt.Token) (interface{}, error) {
		if token.Method != golangjwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %s", token.Method.Alg())
		}
		return []byte(setting.AppConfig.JwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
