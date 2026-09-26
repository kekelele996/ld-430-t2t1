package service

import (
	"testing"

	"github.com/assethub/assethub/internal/constants"
	"github.com/assethub/assethub/internal/model"
)

func TestHasScope(t *testing.T) {
	t.Parallel()
	credential := &model.TeamCredential{
		Scopes: []string{
			string(constants.CredentialScopeAssetsRead),
			string(constants.CredentialScopeCollectionsWrite),
		},
	}
	tests := []struct {
		name  string
		scope constants.CredentialScope
		want  bool
	}{
		{name: "granted read", scope: constants.CredentialScopeAssetsRead, want: true},
		{name: "granted collections", scope: constants.CredentialScopeCollectionsWrite, want: true},
		{name: "missing download", scope: constants.CredentialScopeDownloadsWrite, want: false},
		{name: "unknown scope", scope: constants.CredentialScope("assets:delete"), want: false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := HasScope(credential, tt.scope); got != tt.want {
				t.Fatalf("HasScope(%q) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}
