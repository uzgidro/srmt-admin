package token

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"srmt-admin/internal/lib/model/user"
	"time"
)

// Определяем кастомные ошибки, которые будет возвращать наш сервис.
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token has expired")
)

// Pair содержит пару токенов: доступа и обновления.
type Pair struct {
	AccessToken  string
	RefreshToken string
}

// Claims — это полезная нагрузка, которую мы храним в токене.
type Claims struct {
	jwt.RegisteredClaims
	UserID int64    `json:"uid"`
	Name   string   `json:"name"`
	Roles  []string `json:"roles"`
}

// Token — это наш сервис для работы с JWT.
// Он не содержит логгера и возвращает чистые ошибки.
type Token struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// New создает новый экземпляр JWT сервиса.
// Принимает разные секреты и TTL для access и refresh токенов.
func New(
	secret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) (*Token, error) {
	if secret == "" {
		return nil, errors.New("jwt secrets are required")
	}
	return &Token{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

// Create создает новую пару access и refresh токенов для пользователя.
func (s *Token) Create(u *user.Model) (Pair, error) {
	accessToken, err := s.createAccessToken(u)
	if err != nil {
		return Pair{}, errors.New("failed to create access token")
	}

	refreshToken, err := s.createRefreshToken(u)
	if err != nil {
		return Pair{}, errors.New("failed to create refresh token")
	}

	return Pair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Verify проверяет токен доступа.
func (s *Token) Verify(token string) (*Claims, error) {
	return s.verifyToken(token)
}

func (s *Token) GetRefreshTTL() time.Duration {
	return s.refreshTTL
}

// createToken — это внутренний хелпер для создания токена.
func (s *Token) createAccessToken(u *user.Model) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: u.ID,
		Name:   u.FIO,
		Roles:  u.Roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.secret)
}

func (s *Token) createRefreshToken(u *user.Model) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: u.ID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.secret)
}

// verifyToken — это внутренний хелпер для проверки токена.
func (s *Token) verifyToken(tokenString string) (*Claims, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken // Неожиданный алгоритм подписи
		}
		return s.secret, nil
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, keyFunc)
	if err != nil {
		// Преобразуем ошибку библиотеки в нашу кастомную ошибку
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
