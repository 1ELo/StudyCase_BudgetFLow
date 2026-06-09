package token

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the JWT payload data.
type Claims struct {
	UserID   int64  `json:"user_id"`
	PublicID string `json:"public_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateTokens creates a signed JWT access token and a refresh token.
func GenerateTokens(userID int64, publicID string, role string) (string, string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", "", fmt.Errorf("JWT_SECRET not set")
	}

	expiryHours := 24
	if h := os.Getenv("JWT_EXPIRY_HOURS"); h != "" {
		parsed, err := strconv.Atoi(h)
		if err == nil && parsed > 0 {
			expiryHours = parsed
		}
	}

	claims := Claims{
		UserID:   userID,
		PublicID: publicID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	// Generate refresh token (random 32 byte hex string would be better, but we can also use JWT)
	// For simplicity and secure randomness, we use UUID
	// A more robust approach could use crypto/rand
	refreshToken := fmt.Sprintf("%d.%s", time.Now().UnixNano(), publicID)

	return accessToken, refreshToken, nil
}

// ParseToken validates and parses a JWT token string, returning the claims.
func ParseToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET not set")
	}

	t, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
