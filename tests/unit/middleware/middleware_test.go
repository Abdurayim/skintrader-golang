package middleware_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
	jwtpkg "skintrader-go/internal/pkg/jwt"
	"skintrader-go/tests/mocks"
	"skintrader-go/tests/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestJWTManager() *jwtpkg.Manager {
	return jwtpkg.NewManager("test-secret-key-middleware", 15*time.Minute, 7*24*time.Hour)
}

// ---------------------------------------------------------------------------
// extractBearerToken tests (tested indirectly via AuthenticateUser)
// ---------------------------------------------------------------------------

func TestExtractBearerToken_ValidBearer(t *testing.T) {
	user := testutil.NewTestUser()
	jwtManager := newTestJWTManager()

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return user, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	tokens, _ := jwtManager.GenerateUserTokens(user.ID)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusOK)
}

func TestExtractBearerToken_MissingHeader(t *testing.T) {
	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

func TestExtractBearerToken_EmptyBearer(t *testing.T) {
	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	r.ServeHTTP(w, req)

	// An empty bearer token should fail validation.
	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

func TestExtractBearerToken_NotBearerScheme(t *testing.T) {
	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

func TestExtractBearerToken_InvalidToken(t *testing.T) {
	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

func TestExtractBearerToken_ExpiredToken(t *testing.T) {
	expiredManager := jwtpkg.NewManager("test-secret-key-middleware", 1*time.Millisecond, 1*time.Millisecond)
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(expiredManager, userRepo, adminRepo)

	userID := uuid.New()
	tokens, _ := expiredManager.GenerateUserTokens(userID)

	time.Sleep(10 * time.Millisecond)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

func TestAuthenticateUser_BannedUser(t *testing.T) {
	jwtManager := newTestJWTManager()
	bannedUser := testutil.NewTestUser()
	bannedUser.Status = domain.UserStatusBanned

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return bannedUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	tokens, _ := jwtManager.GenerateUserTokens(bannedUser.ID)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusForbidden)
}

func TestAuthenticateUser_UserNotFound(t *testing.T) {
	jwtManager := newTestJWTManager()

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	tokens, _ := jwtManager.GenerateUserTokens(uuid.New())

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

func TestAuthenticateUser_AdminTokenShouldFail(t *testing.T) {
	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	// Generate admin tokens and try to use with user endpoint.
	adminID := uuid.New()
	tokens, _ := jwtManager.GenerateAdminTokens(adminID)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/test", authMw.AuthenticateUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// RequireKYC tests
// ---------------------------------------------------------------------------

func TestRequireKYC_Verified(t *testing.T) {
	user := testutil.NewTestUser()
	user.KYCStatus = domain.KYCStatusVerified

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}, authMw.RequireKYC(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusOK)
}

func TestRequireKYC_NotVerified(t *testing.T) {
	user := testutil.NewTestUser()
	user.KYCStatus = domain.KYCStatusNotSubmitted

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}, authMw.RequireKYC(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusForbidden)
}

func TestRequireKYC_Pending(t *testing.T) {
	user := testutil.NewTestUser()
	user.KYCStatus = domain.KYCStatusPending

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}, authMw.RequireKYC(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusForbidden)
}

func TestRequireKYC_NoUser(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", authMw.RequireKYC(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// RequireActiveSubscription tests
// ---------------------------------------------------------------------------

func TestRequireActiveSubscription_Active(t *testing.T) {
	user := testutil.NewTestUser()
	user.SubscriptionStatus = domain.SubscriptionStatusActive

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}, authMw.RequireActiveSubscription(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusOK)
}

func TestRequireActiveSubscription_GracePeriod(t *testing.T) {
	user := testutil.NewTestUser()
	user.SubscriptionStatus = domain.SubscriptionStatusGracePeriod
	graceEnd := time.Now().Add(72 * time.Hour)
	user.GracePeriodEndsAt = &graceEnd

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}, authMw.RequireActiveSubscription(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusOK)

	// Check grace period headers.
	if w.Header().Get("X-Grace-Period-Warning") != "true" {
		t.Fatal("expected X-Grace-Period-Warning header to be 'true'")
	}
	if w.Header().Get("X-Grace-Period-Ends") == "" {
		t.Fatal("expected X-Grace-Period-Ends header to be set")
	}
}

func TestRequireActiveSubscription_None(t *testing.T) {
	user := testutil.NewTestUser()
	user.SubscriptionStatus = domain.SubscriptionStatusNone

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}, authMw.RequireActiveSubscription(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusForbidden)
}

func TestRequireActiveSubscription_Expired(t *testing.T) {
	user := testutil.NewTestUser()
	user.SubscriptionStatus = domain.SubscriptionStatusExpired

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}, authMw.RequireActiveSubscription(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusForbidden)
}

func TestRequireActiveSubscription_NoUser(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", authMw.RequireActiveSubscription(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// AuthenticateAdmin tests
// ---------------------------------------------------------------------------

func TestAuthenticateAdmin_Success(t *testing.T) {
	jwtManager := newTestJWTManager()
	admin := testutil.NewTestAdmin()

	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Admin, error) {
			return admin, nil
		},
	}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	tokens, _ := jwtManager.GenerateAdminTokens(admin.ID)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/admin/test", authMw.AuthenticateAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusOK)
}

func TestAuthenticateAdmin_DeactivatedAdmin(t *testing.T) {
	jwtManager := newTestJWTManager()
	admin := testutil.NewTestAdmin()
	admin.IsActive = false

	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Admin, error) {
			return admin, nil
		},
	}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	tokens, _ := jwtManager.GenerateAdminTokens(admin.ID)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/admin/test", authMw.AuthenticateAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusForbidden)
}

func TestAuthenticateAdmin_UserTokenShouldFail(t *testing.T) {
	jwtManager := newTestJWTManager()

	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	tokens, _ := jwtManager.GenerateUserTokens(uuid.New())

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.GET("/admin/test", authMw.AuthenticateAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/admin/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	r.ServeHTTP(w, req)

	testutil.AssertEqual(t, w.Code, http.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// RequirePermission tests
// ---------------------------------------------------------------------------

func TestRequirePermission_SuperAdmin(t *testing.T) {
	admin := testutil.NewTestAdmin()
	admin.Role = domain.AdminRoleSuperAdmin

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("admin", admin)
		c.Next()
	}, authMw.RequirePermission("manage_users"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusOK)
}

func TestRequirePermission_InsufficientPermissions(t *testing.T) {
	admin := testutil.NewTestAdmin()
	admin.Role = domain.AdminRoleSupport
	admin.Permissions = domain.RolePermissions[domain.AdminRoleSupport]

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("admin", admin)
		c.Next()
	}, authMw.RequirePermission("manage_users"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusForbidden)
}

// ---------------------------------------------------------------------------
// RequireSuperAdmin tests
// ---------------------------------------------------------------------------

func TestRequireSuperAdmin_IsSuperAdmin(t *testing.T) {
	admin := testutil.NewTestAdmin()
	admin.Role = domain.AdminRoleSuperAdmin

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("admin", admin)
		c.Next()
	}, authMw.RequireSuperAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusOK)
}

func TestRequireSuperAdmin_NotSuperAdmin(t *testing.T) {
	admin := testutil.NewTestAdmin()
	admin.Role = domain.AdminRoleModerator

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	jwtManager := newTestJWTManager()
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	authMw := middleware.NewAuthMiddleware(jwtManager, userRepo, adminRepo)

	r.GET("/test", func(c *gin.Context) {
		c.Set("admin", admin)
		c.Next()
	}, authMw.RequireSuperAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	c.Request, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)

	testutil.AssertEqual(t, w.Code, http.StatusForbidden)
}
