package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kotban/backend/internal/config"
	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepo
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepo, cfg *config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type JWTClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

func (s *AuthService) Register(email, password, name, phone, companyName string) (*model.User, error) {
	// Check if user already exists
	existing, _ := s.userRepo.FindByEmail(email)
	if existing != nil {
		return nil, errors.New("email sudah terdaftar")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("gagal memproses password")
	}

	user := &model.User{
		Email:       email,
		Password:    string(hashedPassword),
		Name:        name,
		Role:        "client",
		Phone:       phone,
		CompanyName: companyName,
		IsActive:    true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, errors.New("gagal membuat akun")
	}

	return user, nil
}

func (s *AuthService) Login(email, password string) (*model.User, *TokenPair, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil || user == nil {
		return nil, nil, errors.New("email atau password salah")
	}

	if !user.IsActive {
		return nil, nil, errors.New("akun tidak aktif")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, nil, errors.New("email atau password salah")
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, nil, errors.New("gagal membuat token")
	}

	// Update last login
	now := time.Now()
	s.userRepo.UpdateLastLogin(user.ID, &now)

	return user, tokens, nil
}

func (s *AuthService) RefreshToken(refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, errors.New("refresh token tidak valid")
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil || user == nil {
		return nil, errors.New("user tidak ditemukan")
	}

	if !user.IsActive {
		return nil, errors.New("akun tidak aktif")
	}

	return s.generateTokenPair(user)
}

func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token tidak valid")
	}

	return claims, nil
}

func (s *AuthService) GetUserByID(id uint) (*model.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *AuthService) generateTokenPair(user *model.User) (*TokenPair, error) {
	// Access token
	accessExpiry := time.Now().Add(time.Duration(s.cfg.JWTExpireHours) * time.Hour)
	accessClaims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "kotban-api",
			Subject:   user.Email,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshExpiry := time.Now().Add(time.Duration(s.cfg.JWTRefreshExpHours) * time.Hour)
	refreshClaims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "kotban-api-refresh",
			Subject:   user.Email,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshString, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessString,
		RefreshToken: refreshString,
		ExpiresAt:    accessExpiry.Unix(),
	}, nil
}
