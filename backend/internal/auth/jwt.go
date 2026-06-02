package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const DefaultTokenTTL = 24 * time.Hour

type TokenConfig struct {
	Secret   string
	Issuer   string
	Audience string
	TTL      time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(cfg TokenConfig, userID, role string) (string, error) {
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTokenTTL
	}

	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{cfg.Audience},
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.TTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}

func ParseToken(cfg TokenConfig, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method")
		}
		return []byte(cfg.Secret), nil
	},
		jwt.WithIssuer(cfg.Issuer),
		jwt.WithAudience(cfg.Audience),
		jwt.WithIssuedAt(),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.ID == "" {
		return nil, errors.New("missing token id")
	}
	if claims.IssuedAt == nil {
		return nil, errors.New("missing issued at")
	}
	if claims.NotBefore == nil {
		return nil, errors.New("missing not before")
	}
	if claims.Subject == "" || claims.Subject != claims.UserID {
		return nil, errors.New("invalid subject")
	}
	return claims, nil
}
