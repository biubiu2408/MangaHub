package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

type SignedDetails struct {
	UserId   int64
	Username string
	Role     string
	jwt.RegisteredClaims
}

func GenerateJWT(userID int64, username string, role string) (string, error) {
	claims := &SignedDetails{
		UserId:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "MangaHub",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// return signedToken, nil
func GetAccessToken(c *gin.Context) (string, error) {
	//try cookie first
	tokenString, err := c.Cookie("jwt")
	if err == nil && tokenString != "" {
		return tokenString, nil
	}

	//try get header
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}
	tokenString = authHeader[len("Bearer "):]
	if tokenString == "" {
		return "", errors.New("access token is required")
	}
	return tokenString, nil
}

func ValidateToken(tokenString string) (*SignedDetails, error) {
	claims := &SignedDetails{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(jwtKey), nil
	})
	if err != nil {
		return nil, err
	}

	//security check
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, errors.New("unexpected signing method")
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}
func GetUserIdFromContext(c *gin.Context) (int64, error) {
	userId, exists := c.Get("user_id")
	if !exists {
		return 0, errors.New("userId does not exist in this context")
	}

	switch v := userId.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case string:
		// fallback if you ever stored it as string
		parsed, err := strconv.ParseInt(v, 10, 64)
		fmt.Printf("User ID after parsed: %v, exists: %v\n", parsed, exists)

		if err != nil {
			return 0, errors.New("invalid userId format")
		}
		return parsed, nil
	default:
		return 0, errors.New("unexpected type for userId")
	}
}
func GetUserNameFromContext(c *gin.Context) (string, error) {
	username, exists := c.Get("username")
	if !exists {
		return "", errors.New("username does not exist in this context")
	}

	switch v := username.(type) {
	case string:
		return v, nil
	default:
		return "", errors.New("unexpected type for username")
	}
}
func GetUserRoleFromContext(c *gin.Context) (string, error) {
	role, exists := c.Get("role")
	if !exists {
		return "", errors.New("role does not exist in this context")
	}
	switch v := role.(type) {
	case string:
		return v, nil
	default:
		return "", errors.New("unexpected type for role")
	}
}
func ClearUserContext(c *gin.Context) {
	c.Set("user_id", nil)
	c.Set("username", nil)
	c.Set("role", nil)
}

func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mangahub", "token"), nil
}

func SaveToken(token string) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}

	// ensure folder exists
	os.MkdirAll(filepath.Dir(path), 0700)

	return os.WriteFile(path, []byte(token), 0600)
}

func LoadToken() (string, error) {
	path, err := tokenFilePath()
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func ClearToken() error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func DeviceID() string {
	return fmt.Sprintf("device-%d", time.Now().UnixNano())
}
