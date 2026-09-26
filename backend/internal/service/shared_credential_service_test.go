package service

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/model"
	"github.com/assethub/assethub/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func newTestSharedService() *SharedCredentialService {
	return &SharedCredentialService{
		credRepo: &fakeCredStore{},
		logRepo:  &fakeCallLogStore{},
		redis:    &fakeQuotaCounter{},
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestSharedCredentialCreateReturnsPlaintextOnce(t *testing.T) {
	t.Parallel()
	svc := newTestSharedService()
	cred, plain, err := svc.Create(context.Background(), "design-partner",
		[]string{string(constants.ScopeAssetsRead), string(constants.ScopeAssetsRead)}, 100, nil, primitive.NewObjectID())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if plain == "" || cred.KeyHash == plain {
		t.Fatal("plaintext key must be returned once and never equal the stored hash")
	}
	if len(cred.Scopes) != 1 {
		t.Fatalf("duplicate scopes should be de-duplicated, got %v", cred.Scopes)
	}
	if cred.KeyLast4 != plain[len(plain)-4:] {
		t.Fatal("stored last4 must match the plaintext key tail")
	}
}

func TestSharedCredentialCreateRejectsBadInput(t *testing.T) {
	t.Parallel()
	svc := newTestSharedService()

	if _, _, err := svc.Create(context.Background(), "x", []string{"bogus:scope"}, 10, nil, primitive.NewObjectID()); err == nil {
		t.Fatal("expected error for invalid scope")
	}
	past := time.Now().Add(-time.Hour)
	if _, _, err := svc.Create(context.Background(), "x", []string{string(constants.ScopeAssetsRead)}, 10, &past, primitive.NewObjectID()); err == nil {
		t.Fatal("expected error for past expires_at")
	}
}

func TestSharedCredentialAuthenticate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := newTestSharedService()
	cred, plain, err := svc.Create(ctx, "partner", []string{string(constants.ScopeAssetsRead)}, 5, nil, primitive.NewObjectID())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.Authenticate(ctx, plain)
	if err != nil {
		t.Fatalf("Authenticate() valid key error = %v", err)
	}
	if got.ID != cred.ID {
		t.Fatal("Authenticate() resolved the wrong credential")
	}

	if _, err := svc.Authenticate(ctx, "ahk_does_not_exist"); err == nil {
		t.Fatal("unknown key must be rejected")
	}

	if err := svc.Revoke(ctx, cred.ID.Hex()); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if _, err := svc.Authenticate(ctx, plain); err == nil {
		t.Fatal("revoked credential must be rejected immediately")
	}
}

func TestSharedCredentialAuthenticateExpired(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := newTestSharedService()
	cred, plain, err := svc.Create(ctx, "partner", []string{string(constants.ScopeAssetsRead)}, 5, nil, primitive.NewObjectID())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	// Simulate time passing past the expiry date by planting an expired record.
	expiry := time.Now().Add(-time.Minute)
	cred.ExpiresAt = &expiry
	if err := (svc.credRepo.(*fakeCredStore)).save(ctx, cred); err != nil {
		t.Fatalf("save expired credential: %v", err)
	}
	if _, err := svc.Authenticate(ctx, plain); err == nil {
		t.Fatal("expired credential must be rejected with 401")
	}
}

func TestConsumeDailyQuota(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := newTestSharedService()
	id := primitive.NewObjectID()

	for i := int64(1); i <= 3; i++ {
		if err := svc.ConsumeDailyQuota(ctx, id, 3); err != nil {
			t.Fatalf("ConsumeDailyQuota() call %d error = %v", i, err)
		}
	}
	if err := svc.ConsumeDailyQuota(ctx, id, 3); err == nil {
		t.Fatal("4th call must be rejected once the daily limit is reached")
	}
}

// ---- fakes ----

type fakeCredStore struct {
	byHash map[string]*model.SharedCredential
}

func (f *fakeCredStore) Create(_ context.Context, c *model.SharedCredential) error {
	if f.byHash == nil {
		f.byHash = map[string]*model.SharedCredential{}
	}
	if c.ID.IsZero() {
		c.ID = primitive.NewObjectID()
	}
	copy := *c
	f.byHash[c.KeyHash] = &copy
	return nil
}

func (f *fakeCredStore) save(_ context.Context, c *model.SharedCredential) error {
	if f.byHash == nil {
		f.byHash = map[string]*model.SharedCredential{}
	}
	copy := *c
	f.byHash[c.KeyHash] = &copy
	return nil
}

func (f *fakeCredStore) FindByHash(_ context.Context, hash string) (*model.SharedCredential, error) {
	if c, ok := f.byHash[hash]; ok {
		copy := *c
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (f *fakeCredStore) FindByID(_ context.Context, id primitive.ObjectID) (*model.SharedCredential, error) {
	for _, c := range f.byHash {
		if c.ID == id {
			copy := *c
			return &copy, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeCredStore) List(_ context.Context) ([]model.SharedCredential, error) {
	out := make([]model.SharedCredential, 0, len(f.byHash))
	for _, c := range f.byHash {
		out = append(out, *c)
	}
	return out, nil
}

func (f *fakeCredStore) Revoke(_ context.Context, id primitive.ObjectID) error {
	for _, c := range f.byHash {
		if c.ID == id {
			if c.Revoked {
				return repository.ErrNotFound
			}
			c.Revoked = true
			now := time.Now()
			c.RevokedAt = &now
			return nil
		}
	}
	return repository.ErrNotFound
}

type fakeCallLogStore struct {
	entries []model.SharedCallLog
}

func (f *fakeCallLogStore) Create(_ context.Context, e *model.SharedCallLog) error {
	if e.ID.IsZero() {
		e.ID = primitive.NewObjectID()
	}
	f.entries = append(f.entries, *e)
	return nil
}

func (f *fakeCallLogStore) ListByCredential(_ context.Context, id primitive.ObjectID, _, _ int64) ([]model.SharedCallLog, error) {
	var out []model.SharedCallLog
	for _, e := range f.entries {
		if e.CredentialID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

type fakeQuotaCounter struct {
	counts map[string]int64
}

func (f *fakeQuotaCounter) IncrWithTTL(_ context.Context, key string, _ time.Duration) (int64, error) {
	if f.counts == nil {
		f.counts = map[string]int64{}
	}
	f.counts[key]++
	return f.counts[key], nil
}
