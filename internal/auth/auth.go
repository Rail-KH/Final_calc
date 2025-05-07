package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	Secret = "calculator_by_Rail-KH"
)

type User struct {
	UserID   int
	Login    string
	Password string
}

func HashPass(pass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), 14)
	return string(hash), err
}

func CheckCorPass(pass, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)) == nil
}

func GenJWT(ID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"UserID": ID,
	})
	return token.SignedString([]byte(Secret))
}

func ParseJWT(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(Secret), nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		ID := int(claims["UserID"].(float64))
		return ID, nil
	}
	return 0, errors.New("invalid data")
}
