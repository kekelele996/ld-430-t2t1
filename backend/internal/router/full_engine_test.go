package router

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/assethub/assethub/internal/client"
	"github.com/assethub/assethub/internal/config"
	"github.com/assethub/assethub/internal/middleware"
	"github.com/assethub/assethub/internal/util"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestFullEngineRegistersSharedRoutes builds the complete engine (admin
// credential routes + external shared routes coexisting under /api/v1 and /v1)
// to guarantee the radix tree accepts the combined wiring at startup.
func TestFullEngineRegistersSharedRoutes(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("router.New panicked: %v", r)
		}
	}()

	cfg := &config.Config{
		AppEnv: "test",
		Port:   "8080",
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
	}
	jwtManager := util.NewJWTManager("test-secret-at-least-16-chars", "assethub-test", time.Hour)
	shared := &middleware.SharedCredentialServices{}

	// A disconnected mongo handle and a non-connecting rate limiter are enough to
	// exercise route registration (no request is served), proving the radix tree
	// accepts the combined wiring.
	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("mongo connect handle: %v", err)
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()
	dbHandle := &client.MongoClient{Client: mongoClient, DB: mongoClient.Database("route_registration_test")}
	rateLimiter := middleware.NewRateLimiter(nil, slog.Default())

	engine := New(Handlers{}, cfg, jwtManager, rateLimiter, dbHandle, nil, shared, slog.Default())

	want := map[string]bool{
		"POST-/api/v1/team-credentials":                         false,
		"GET-/api/v1/shared/assets":                             false,
		"POST-/api/v1/shared/assets/:id/download":               false,
		"DELETE-/api/v1/shared/collections/:id/assets/:assetId": false,
		"GET-/v1/shared/assets":                                 false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + "-" + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("expected route %s registered in full engine", route)
		}
	}
}
