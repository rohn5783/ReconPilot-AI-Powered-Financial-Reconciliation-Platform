package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"reconpilot/internal/domain"
	"reconpilot/internal/repository"
)

var (
	ErrUserAlreadyExists  = errors.New("a user with this email address already exists")
	ErrInvalidCredentials = errors.New("invalid email address or password")
	ErrUserNotFound       = errors.New("user account not found")
	ErrInvalidToken       = errors.New("invalid or expired authentication token")
)

// AuthClaims holds standard JWT claims along with user identification metadata.
type AuthClaims struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	FullName    string    `json:"full_name"`
	CompanyName string    `json:"company_name"`
	jwt.RegisteredClaims
}

// AuthService orchestrates user registration, password verification, and JWT session handling.
type AuthService struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

// NewAuthService instantiates an AuthService with user storage and secret.
func NewAuthService(userRepo repository.UserRepository, jwtSecret string) *AuthService {
	if strings.TrimSpace(jwtSecret) == "" {
		jwtSecret = "reconpilot-jwt-insecure-dev-secret-change-in-prod"
	}
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  7 * 24 * time.Hour, // 7 days token lifetime
	}
}

// Register creates a new user profile with securely hashed password and returns an auth session.
func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	if len(req.Password) < 6 {
		return nil, errors.New("password must be at least 6 characters")
	}

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		parts := strings.Split(email, "@")
		fullName = parts[0]
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "FINANCE_ANALYST"
	}

	user := domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashBytes),
		FullName:     fullName,
		CompanyName:  strings.TrimSpace(req.CompanyName),
		Role:         role,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, &user); err != nil {
		return nil, fmt.Errorf("persist user: %w", err)
	}

	token, err := s.generateToken(&user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &domain.AuthResponse{
		Token: token,
		User:  user.ToPublic(),
	}, nil
}

// Login authenticates user credentials and produces a JWT session.
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &domain.AuthResponse{
		Token: token,
		User:  user.ToPublic(),
	}, nil
}

// GetCurrentUser returns the sanitized user profile for the specified user ID.
func (s *AuthService) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*domain.UserPublic, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	pub := user.ToPublic()
	return &pub, nil
}

// ValidateToken parses and validates a signed JWT token string.
func (s *AuthService) ValidateToken(tokenString string) (*AuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*AuthClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) generateToken(user *domain.User) (string, error) {
	now := time.Now().UTC()
	claims := AuthClaims{
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role,
		FullName:    user.FullName,
		CompanyName: user.CompanyName,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			Issuer:    "reconpilot-engine",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
