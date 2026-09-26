package middleware

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/errors"
	"github.com/assethub/assethub/internal/model"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func businessUnauthorized(msg string) error {
	return errors.NewBusinessError(constants.CodeUnauthorized, msg)
}

func businessRateLimited(msg string) error {
	return errors.NewBusinessError(constants.CodeRateLimited, msg)
}

type fakeSharedAuthenticator struct {
	credential *model.SharedCredential
	err        error
	calls      int
}

func (f *fakeSharedAuthenticator) Authenticate(_ context.Context, _ string) (*model.SharedCredential, error) {
	f.calls++
	return f.credential, f.err
}

type fakeSharedQuota struct {
	err error
}

func (f *fakeSharedQuota) ConsumeDailyQuota(_ context.Context, _ primitive.ObjectID, _ int64) error {
	return f.err
}

type fakeSharedRecorder struct {
	entries []model.SharedCallLog
}

func (f *fakeSharedRecorder) RecordCall(_ context.Context, e *model.SharedCallLog) error {
	f.entries = append(f.entries, *e)
	return nil
}

func newSharedCredential(scopes []string) *model.SharedCredential {
	return &model.SharedCredential{
		ID:         primitive.NewObjectID(),
		Name:       "partner",
		Scopes:     scopes,
		DailyLimit: 2,
	}
}

func buildSharedRouter(auth SharedAuthenticator, quota SharedQuotaConsumer, recorder SharedCallRecorder) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	shared := r.Group("/shared")
	shared.Use(
		SharedCallLogger(recorder, slog.New(slog.NewTextHandler(io.Discard, nil))),
		SharedAuth(auth),
	)
	read := shared.Group("/assets")
	read.Use(RequireSharedScope(constants.ScopeAssetsRead), SharedDailyQuota(quota))
	read.GET("/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	download := shared.Group("/downloads")
	download.Use(RequireSharedScope(constants.ScopeDownloadsWrite), SharedDailyQuota(quota))
	download.POST("", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return r
}

func doSharedRequest(r *gin.Engine, method, path string, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if key != "" {
		req.Header.Set(ContextSharedKeyHeader, key)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSharedAuthMissingKeyReturns401(t *testing.T) {
	t.Parallel()
	auth := &fakeSharedAuthenticator{err: businessUnauthorized(constants.MsgSharedCredentialInvalid)}
	rec := &fakeSharedRecorder{}
	r := buildSharedRouter(auth, &fakeSharedQuota{}, rec)

	w := doSharedRequest(r, http.MethodGet, "/shared/assets/x", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if len(rec.entries) != 0 {
		t.Fatal("unknown keys must not be recorded")
	}
}

func TestSharedAuthExpiredOrRevokedReturns401AndLogs(t *testing.T) {
	t.Parallel()
	cred := newSharedCredential([]string{string(constants.ScopeAssetsRead)})
	auth := &fakeSharedAuthenticator{
		credential: cred,
		err:        businessUnauthorized(constants.MsgSharedCredentialRevoked),
	}
	rec := &fakeSharedRecorder{}
	r := buildSharedRouter(auth, &fakeSharedQuota{}, rec)

	w := doSharedRequest(r, http.MethodGet, "/shared/assets/x", "ahk_revoked")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
	if len(rec.entries) != 1 || rec.entries[0].StatusCode != http.StatusUnauthorized {
		t.Fatal("revoked call must be logged with 401")
	}
}

func TestSharedScopeForbiddenReturns403(t *testing.T) {
	t.Parallel()
	// Credential may read assets but not register downloads.
	cred := newSharedCredential([]string{string(constants.ScopeAssetsRead)})
	auth := &fakeSharedAuthenticator{credential: cred}
	quota := &fakeSharedQuota{}
	rec := &fakeSharedRecorder{}
	r := buildSharedRouter(auth, quota, rec)

	w := doSharedRequest(r, http.MethodPost, "/shared/downloads", "ahk_valid")
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if len(rec.entries) != 1 || rec.entries[0].Result != "forbidden" {
		t.Fatal("out-of-scope call must be logged as forbidden")
	}

	// In-scope request succeeds and out-of-scope attempts must not consume quota.
	ok := doSharedRequest(r, http.MethodGet, "/shared/assets/x", "ahk_valid")
	if ok.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", ok.Code)
	}
}

func TestSharedQuotaExhaustedReturns429(t *testing.T) {
	t.Parallel()
	cred := newSharedCredential([]string{string(constants.ScopeAssetsRead)})
	auth := &fakeSharedAuthenticator{credential: cred}
	quota := &fakeSharedQuota{err: businessRateLimited(constants.MsgSharedQuotaExhausted)}
	rec := &fakeSharedRecorder{}
	r := buildSharedRouter(auth, quota, rec)

	w := doSharedRequest(r, http.MethodGet, "/shared/assets/x", "ahk_valid")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", w.Code)
	}
	if len(rec.entries) != 1 || rec.entries[0].Result != "quota_exhausted" {
		t.Fatal("quota-exhausted call must be logged accordingly")
	}
}

func TestSharedSuccessRecordsCall(t *testing.T) {
	t.Parallel()
	cred := newSharedCredential([]string{string(constants.ScopeAssetsRead)})
	auth := &fakeSharedAuthenticator{credential: cred}
	rec := &fakeSharedRecorder{}
	r := buildSharedRouter(auth, &fakeSharedQuota{}, rec)

	w := doSharedRequest(r, http.MethodGet, "/shared/assets/abc", "ahk_valid")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(rec.entries) != 1 {
		t.Fatalf("recorded %d entries, want 1", len(rec.entries))
	}
	entry := rec.entries[0]
	if entry.CredentialID != cred.ID || entry.Path != "/shared/assets/abc" || entry.StatusCode != 200 || entry.Result != "success" {
		t.Fatalf("unexpected call log entry: %+v", entry)
	}
}
