package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/assethub/assethub/internal/constants"
	apperrors "github.com/assethub/assethub/internal/errors"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type fakeCredentialServices struct {
	credential *model.TeamCredential
	authErr    error
	exceeded   bool
	logged     int
	entry      *model.CredentialAccessLog
}

func (f *fakeCredentialServices) Authenticate(_ context.Context, _ string) (*model.TeamCredential, error) {
	if f.authErr != nil {
		return nil, f.authErr
	}
	return f.credential, nil
}

func (f *fakeCredentialServices) ConsumeQuota(_ context.Context, _ *model.TeamCredential) (bool, error) {
	return f.exceeded, nil
}

func (f *fakeCredentialServices) TouchLastUsed(_ context.Context, _ *model.TeamCredential) {}

func (f *fakeCredentialServices) LogAccess(_ context.Context, entry *model.CredentialAccessLog) {
	f.logged++
	f.entry = entry
}

func buildSharedEngine(svc *fakeCredentialServices, scope constants.CredentialScope) *gin.Engine {
	gin.SetMode(gin.TestMode)
	g := gin.New()
	protected := g.Group("/s")
	protected.Use(CredentialAccessLog(svc))
	protected.Use(SharedAuth(svc))
	protected.Use(RequireScope(scope))
	protected.Use(CredentialQuota(svc))
	protected.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return g
}

func activeCredential(scopes []string) *model.TeamCredential {
	return &model.TeamCredential{
		ID:     primitive.NewObjectID(),
		Name:   "ci",
		Scopes: scopes,
	}
}

func TestSharedAuthAndScopeAndQuota(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		key        string
		credential *model.TeamCredential
		authErr    error
		exceeded   bool
		scope      constants.CredentialScope
		wantStatus int
	}{
		{name: "missing key", scope: constants.CredentialScopeAssetsRead, wantStatus: http.StatusUnauthorized},
		{
			name:       "invalid key",
			key:        "ahk_bad",
			authErr:    apperrors.NewBusinessError(constants.CodeUnauthorized, constants.MsgCredentialInvalid),
			scope:      constants.CredentialScopeAssetsRead,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "revoked",
			key:        "ahk_x",
			authErr:    apperrors.NewBusinessError(constants.CodeUnauthorized, constants.MsgCredentialRevoked),
			scope:      constants.CredentialScopeAssetsRead,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "expired",
			key:        "ahk_x",
			authErr:    apperrors.NewBusinessError(constants.CodeUnauthorized, constants.MsgCredentialExpired),
			scope:      constants.CredentialScopeAssetsRead,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing scope returns 403",
			key:        "ahk_x",
			credential: activeCredential([]string{string(constants.CredentialScopeAssetsRead)}),
			scope:      constants.CredentialScopeDownloadsWrite,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "quota exhausted returns 429",
			key:        "ahk_x",
			credential: activeCredential([]string{string(constants.CredentialScopeAssetsRead)}),
			scope:      constants.CredentialScopeAssetsRead,
			exceeded:   true,
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:       "authorized returns 200",
			key:        "ahk_x",
			credential: activeCredential([]string{string(constants.CredentialScopeAssetsRead)}),
			scope:      constants.CredentialScopeAssetsRead,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := &fakeCredentialServices{credential: tt.credential, authErr: tt.authErr, exceeded: tt.exceeded}
			engine := buildSharedEngine(svc, tt.scope)

			req := httptest.NewRequest(http.MethodGet, "/s/ok", nil)
			if tt.key != "" {
				req.Header.Set("X-Api-Key", tt.key)
			}
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if svc.logged != 1 {
				t.Fatalf("access log entries = %d, want 1", svc.logged)
			}
		})
	}
}

// TestAccessLogRecordsCredentialPathResultTime verifies the access log entry
// captures the credential, path, result and timestamp of a successful call.
func TestAccessLogRecordsCredentialPathResultTime(t *testing.T) {
	t.Parallel()
	cred := activeCredential([]string{string(constants.CredentialScopeAssetsRead)})
	svc := &fakeCredentialServices{credential: cred}
	engine := buildSharedEngine(svc, constants.CredentialScopeAssetsRead)

	key, err := util.GenerateAPIKey()
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/s/ok?x=1", nil)
	req.Header.Set("X-Api-Key", key.Full)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if svc.entry == nil {
		t.Fatal("expected an access log entry")
	}
	if svc.entry.CredentialID != cred.ID {
		t.Fatalf("logged credential id = %v, want %v", svc.entry.CredentialID, cred.ID)
	}
	if svc.entry.Path != "/s/ok" {
		t.Fatalf("logged path = %q, want /s/ok", svc.entry.Path)
	}
	if svc.entry.Method != http.MethodGet {
		t.Fatalf("logged method = %q, want GET", svc.entry.Method)
	}
	if svc.entry.StatusCode != http.StatusOK || svc.entry.Result != "success" {
		t.Fatalf("logged result = %q/%d, want success/200", svc.entry.Result, svc.entry.StatusCode)
	}
	if svc.entry.CreatedAt.IsZero() {
		t.Fatal("logged timestamp must not be zero")
	}
}
