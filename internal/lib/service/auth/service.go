package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"srmt-admin/internal/lib/model/user"
	"srmt-admin/internal/storage"
	"srmt-admin/internal/token"

	"golang.org/x/crypto/bcrypt"
)

type UserProvider interface {
	GetUserByLogin(ctx context.Context, login string) (*user.Model, string, error)
	GetUserByID(ctx context.Context, id int64) (*user.Model, error)
}

type TokenProvider interface {
	Create(u *user.Model) (token.Pair, error)
	Verify(token string) (*token.Claims, error)
	GetRefreshTTL() time.Duration
}

type Service struct {
	userProvider  UserProvider
	tokenProvider TokenProvider
	log           *slog.Logger
}

func NewService(userProvider UserProvider, tokenProvider TokenProvider, log *slog.Logger) *Service {
	return &Service{
		userProvider:  userProvider,
		tokenProvider: tokenProvider,
		log:           log,
	}
}

func (s *Service) Login(ctx context.Context, login, password string) (token.Pair, time.Duration, error) {
	const op = "service.auth.Login"
	log := s.log.With(slog.String("op", op), slog.String("login", login))

	// 1. Get user
	u, passHash, err := s.userProvider.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return token.Pair{}, 0, storage.ErrUserNotFound
		}
		return token.Pair{}, 0, fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	// 2. Check password
	if err := bcrypt.CompareHashAndPassword([]byte(passHash), []byte(password)); err != nil {
		log.Warn("invalid password")
		return token.Pair{}, 0, storage.ErrInvalidCredentials // Assuming we have this or similar error
	}

	// 3. Check active
	if !u.IsActive {
		log.Warn("user is not active")
		return token.Pair{}, 0, storage.ErrUserDeactivated
	}

	// 4. Create tokens
	pair, err := s.tokenProvider.Create(u)
	if err != nil {
		return token.Pair{}, 0, fmt.Errorf("%s: failed to create tokens: %w", op, err)
	}

	return pair, s.tokenProvider.GetRefreshTTL(), nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (token.Pair, time.Duration, error) {
	const op = "service.auth.Refresh"
	log := s.log.With(slog.String("op", op))

	// 1. Verify token
	claims, err := s.tokenProvider.Verify(refreshToken)
	if err != nil {
		log.Warn("invalid refresh token")
		return token.Pair{}, 0, fmt.Errorf("%s: invalid token: %w", op, err)
	}

	// 2. Get user
	u, err := s.userProvider.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return token.Pair{}, 0, fmt.Errorf("%s: failed to get user: %w", op, err)
	}

	// 3. Create new tokens
	pair, err := s.tokenProvider.Create(u)
	if err != nil {
		return token.Pair{}, 0, fmt.Errorf("%s: failed to create tokens: %w", op, err)
	}

	return pair, s.tokenProvider.GetRefreshTTL(), nil
}
