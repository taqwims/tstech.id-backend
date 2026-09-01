package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tstech/backend/internal/config"
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
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

// FirebaseLogin handles authentication from Google / Firebase on tstech.id portal
func (s *AuthService) FirebaseLogin(email, name, avatarURL, phone string) (*model.User, *TokenPair, error) {
	if email == "" {
		return nil, nil, errors.New("email dari akun Google/Firebase tidak valid")
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil || user == nil {
		// Auto-register new client from Google Sign-In
		if name == "" {
			name = strings.Split(email, "@")[0]
		}
		randomPass := fmt.Sprintf("GAuth_%d_%s", time.Now().UnixNano(), email)
		hashedPass, _ := bcrypt.GenerateFromPassword([]byte(randomPass), bcrypt.DefaultCost)

		user = &model.User{
			Email:     email,
			Password:  string(hashedPass),
			Name:      name,
			AvatarURL: avatarURL,
			Phone:     phone,
			Role:      "client",
			IsActive:  true,
		}

		if err := s.userRepo.Create(user); err != nil {
			return nil, nil, fmt.Errorf("gagal membuat user: %w", err)
		}
	} else {
		if !user.IsActive {
			return nil, nil, errors.New("akun Anda telah dinonaktifkan")
		}
		if avatarURL != "" && user.AvatarURL == "" {
			user.AvatarURL = avatarURL
			_ = s.userRepo.UpdateProfile(user.ID, map[string]interface{}{"avatar_url": avatarURL})
		}
	}

	now := time.Now()
	s.userRepo.UpdateLastLogin(user.ID, &now)

	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, nil, errors.New("gagal membuat session token")
	}

	return user, tokens, nil
}

type SSOVerifyResult struct {
	Valid              bool   `json:"valid"`
	UserID             uint   `json:"user_id"`
	Email              string `json:"email"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	TenantSlug         string `json:"tenant_slug"`
	FullSubdomain      string `json:"full_subdomain"`
	SubscriptionNumber string `json:"subscription_number"`
	PlanCode           string `json:"plan_code"`
	PlanName           string `json:"plan_name"`
	MaxUsers           int    `json:"max_users"`
	MaxStorageGB       int    `json:"max_storage_gb"`
	Issuer             string `json:"iss"`
}

// VerifySSOToken parses and validates a signed SSO JWT from satellite SaaS
func (s *AuthService) VerifySSOToken(tokenString, secretKey string) (*SSOVerifyResult, error) {
	candidates := []string{}
	if secretKey != "" {
		candidates = append(candidates, secretKey)
	}

	// Try reading product_slug or tenant_slug from unverified claims to find product secret
	parser := jwt.NewParser()
	unvToken, _, _ := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if unvToken != nil {
		if unvClaims, ok := unvToken.Claims.(jwt.MapClaims); ok {
			if prodSlug, ok := unvClaims["product_slug"].(string); ok && prodSlug != "" {
				var prod model.SaaSProduct
				query := s.userRepo.GetDB().Where("slug = ?", prodSlug)
				if prodSlug == "school" || prodSlug == "schola" {
					query = s.userRepo.GetDB().Where("slug = ? OR slug = ?", "schola", "school")
				}
				if err := query.First(&prod).Error; err == nil && prod.APISecretKey != "" {
					candidates = append(candidates, prod.APISecretKey)
				}
			}
		}
	}

	candidates = append(candidates, s.cfg.JWTSecret)

	var token *jwt.Token
	var lastErr error

	for _, sec := range candidates {
		t, err := jwt.Parse(tokenString, func(tok *jwt.Token) (interface{}, error) {
			if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("signing method tidak valid")
			}
			return []byte(sec), nil
		})
		if err == nil && t.Valid {
			token = t
			break
		}
		lastErr = err
	}

	if token == nil || !token.Valid {
		errMsg := "SSO token tidak valid atau telah kedaluwarsa"
		if lastErr != nil {
			errMsg = fmt.Sprintf("SSO token tidak valid: %v", lastErr)
		}
		return nil, errors.New(errMsg)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("claims tidak dapat dibaca")
	}

	var uid uint
	if subStr, ok := claims["sub"].(string); ok {
		var idInt int
		fmt.Sscanf(subStr, "%d", &idInt)
		uid = uint(idInt)
	}

	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	role, _ := claims["role"].(string)
	if role == "" {
		role = "admin"
	}
	tenantSlug, _ := claims["tenant_slug"].(string)
	fullSubdomain, _ := claims["full_subdomain"].(string)
	subNumber, _ := claims["subscription_number"].(string)
	if subNumber == "" {
		subNumber, _ = claims["subscription_id"].(string)
	}
	planCode, _ := claims["plan_code"].(string)
	planName, _ := claims["plan_name"].(string)
	iss, _ := claims["iss"].(string)

	var maxUsers int
	if mu, ok := claims["max_users"].(float64); ok {
		maxUsers = int(mu)
	}
	var maxStorage int
	if ms, ok := claims["max_storage_gb"].(float64); ok {
		maxStorage = int(ms)
	}

	return &SSOVerifyResult{
		Valid:              true,
		UserID:             uid,
		Email:              email,
		Name:               name,
		Role:               role,
		TenantSlug:         tenantSlug,
		FullSubdomain:      fullSubdomain,
		SubscriptionNumber: subNumber,
		PlanCode:           planCode,
		PlanName:           planName,
		MaxUsers:           maxUsers,
		MaxStorageGB:       maxStorage,
		Issuer:             iss,
	}, nil
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
			Issuer:    "tstech-api",
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
			Issuer:    "tstech-api-refresh",
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
