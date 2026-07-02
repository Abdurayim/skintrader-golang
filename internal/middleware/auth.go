package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"skintrader-go/internal/domain"
	jwtpkg "skintrader-go/internal/pkg/jwt"
)

type AuthMiddleware struct {
	jwtManager   *jwtpkg.Manager
	userRepo     domain.UserRepository
	adminRepo    domain.AdminRepository
}

func NewAuthMiddleware(jwtManager *jwtpkg.Manager, userRepo domain.UserRepository, adminRepo domain.AdminRepository) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
		userRepo:   userRepo,
		adminRepo:  adminRepo,
	}
}

func (m *AuthMiddleware) AuthenticateUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Authentication required", "code": "UNAUTHORIZED",
			})
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			code := "INVALID_TOKEN"
			if strings.Contains(err.Error(), "expired") {
				code = "TOKEN_EXPIRED"
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Invalid token", "code": code,
			})
			return
		}

		if claims.Role != jwtpkg.RoleUser || claims.Type != jwtpkg.AccessToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Invalid token type", "code": "INVALID_TOKEN",
			})
			return
		}

		user, err := m.userRepo.FindByID(context.Background(), claims.UserID)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "User not found", "code": "UNAUTHORIZED",
			})
			return
		}

		if user.Status == domain.UserStatusBanned {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false, "message": "Account is banned", "code": "ACCOUNT_BANNED",
			})
			return
		}

		if user.Status == domain.UserStatusSuspended {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false, "message": "Account is suspended", "code": "ACCOUNT_SUSPENDED",
			})
			return
		}

		c.Set("user", user)
		c.Set("userId", user.ID)
		c.Next()
	}
}

func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.Next()
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			c.Next()
			return
		}

		if claims.Role != jwtpkg.RoleUser || claims.Type != jwtpkg.AccessToken {
			c.Next()
			return
		}

		user, err := m.userRepo.FindByID(context.Background(), claims.UserID)
		if err != nil || user == nil || user.Status != domain.UserStatusActive {
			c.Next()
			return
		}

		c.Set("user", user)
		c.Set("userId", user.ID)
		c.Next()
	}
}

func (m *AuthMiddleware) RequireKYC() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Authentication required", "code": "UNAUTHORIZED",
			})
			return
		}

		u := user.(*domain.User)
		if u.KYCStatus != domain.KYCStatusVerified {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false, "message": "KYC verification is required", "code": "KYC_REQUIRED",
			})
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireActiveSubscription() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Authentication required", "code": "UNAUTHORIZED",
			})
			return
		}

		u := user.(*domain.User)
		if u.SubscriptionStatus == domain.SubscriptionStatusActive {
			c.Next()
			return
		}

		if u.SubscriptionStatus == domain.SubscriptionStatusGracePeriod {
			if u.GracePeriodEndsAt != nil {
				c.Header("X-Grace-Period-Warning", "true")
				c.Header("X-Grace-Period-Ends", u.GracePeriodEndsAt.Format("2006-01-02T15:04:05Z"))
			}
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false, "message": "Active subscription is required", "code": "SUBSCRIPTION_REQUIRED",
		})
	}
}

func (m *AuthMiddleware) AuthenticateAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Authentication required", "code": "UNAUTHORIZED",
			})
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			code := "INVALID_TOKEN"
			if strings.Contains(err.Error(), "expired") {
				code = "TOKEN_EXPIRED"
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Invalid token", "code": code,
			})
			return
		}

		if claims.Role != jwtpkg.RoleAdmin || claims.Type != jwtpkg.AccessToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Admin access required", "code": "UNAUTHORIZED",
			})
			return
		}

		admin, err := m.adminRepo.FindByID(context.Background(), claims.AdminID)
		if err != nil || admin == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Admin not found", "code": "UNAUTHORIZED",
			})
			return
		}

		if !admin.IsActive {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false, "message": "Admin account is deactivated", "code": "FORBIDDEN",
			})
			return
		}

		c.Set("admin", admin)
		c.Set("adminId", admin.ID)
		c.Next()
	}
}

func (m *AuthMiddleware) RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, exists := c.Get("admin")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Admin access required", "code": "UNAUTHORIZED",
			})
			return
		}

		a := admin.(*domain.Admin)
		if a.Role == domain.AdminRoleSuperAdmin {
			c.Next()
			return
		}

		for _, perm := range permissions {
			if !a.HasPermission(perm) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"success": false, "message": "Insufficient permissions", "code": "FORBIDDEN",
				})
				return
			}
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, exists := c.Get("admin")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false, "message": "Admin access required", "code": "UNAUTHORIZED",
			})
			return
		}

		a := admin.(*domain.Admin)
		if a.Role != domain.AdminRoleSuperAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false, "message": "Super admin access required", "code": "FORBIDDEN",
			})
			return
		}

		c.Next()
	}
}

// GetUserID extracts the authenticated user ID from the context.
func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	id, exists := c.Get("userId")
	if !exists {
		return uuid.Nil, false
	}
	return id.(uuid.UUID), true
}

// GetUser extracts the authenticated user from the context.
func GetUser(c *gin.Context) (*domain.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	return user.(*domain.User), true
}

// GetAdmin extracts the authenticated admin from the context.
func GetAdmin(c *gin.Context) (*domain.Admin, bool) {
	admin, exists := c.Get("admin")
	if !exists {
		return nil, false
	}
	return admin.(*domain.Admin), true
}

func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
