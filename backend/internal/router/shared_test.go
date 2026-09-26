package router

import (
	"testing"
	"time"

	"github.com/assethub/assethub/internal/middleware"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
)

// TestRegisterSharedRoutesNoConflict guards against httprouter radix-tree
// panics caused by registering read (GET) and download (POST) handlers under the
// same /shared/assets prefix with different middleware chains.
func TestRegisterSharedRoutesNoConflict(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registerShared panicked: %v", r)
		}
	}()

	g := gin.New()
	api := g.Group("/api/v1")
	registerShared(api, Handlers{}, &middleware.SharedCredentialServices{})

	wantRoutes := map[string]bool{
		"GET-/api/v1/shared/assets":                             false,
		"GET-/api/v1/shared/assets/:id":                         false,
		"POST-/api/v1/shared/assets/:id/download":               false,
		"POST-/api/v1/shared/collections":                       false,
		"GET-/api/v1/shared/collections":                        false,
		"POST-/api/v1/shared/collections/:id/assets":            false,
		"DELETE-/api/v1/shared/collections/:id/assets/:assetId": false,
	}
	for _, route := range g.Routes() {
		key := route.Method + "-" + route.Path
		if _, ok := wantRoutes[key]; ok {
			wantRoutes[key] = true
		}
	}
	for route, found := range wantRoutes {
		if !found {
			t.Errorf("expected route %s to be registered", route)
		}
	}
}

// TestRegisterTeamCredentialRoutesNoConflict ensures admin credential routes
// register cleanly.
func TestRegisterTeamCredentialRoutesNoConflict(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registerTeamCredential panicked: %v", r)
		}
	}()

	g := gin.New()
	api := g.Group("/api/v1")
	jwtManager := util.NewJWTManager("test-secret-at-least-16-chars", "assethub-test", time.Hour)
	registerTeamCredential(api, Handlers{}, jwtManager)

	if len(g.Routes()) != 5 {
		t.Fatalf("expected 5 team credential routes, got %d", len(g.Routes()))
	}
}
