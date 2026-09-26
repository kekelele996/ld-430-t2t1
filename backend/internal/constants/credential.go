package constants

// CredentialScope enumerates authorization scopes granted to a team credential.
type CredentialScope string

const (
	// CredentialScopeAssetsRead grants read-only access to published assets.
	CredentialScopeAssetsRead CredentialScope = "assets:read"
	// CredentialScopeCollectionsWrite grants maintaining (creating/editing) favorites collections.
	CredentialScopeCollectionsWrite CredentialScope = "collections:write"
	// CredentialScopeDownloadsWrite grants registering download records.
	CredentialScopeDownloadsWrite CredentialScope = "downloads:write"
)

// CredentialStatus enumerates team credential lifecycle states.
type CredentialStatus string

const (
	// CredentialStatusActive means the credential may call shared endpoints.
	CredentialStatusActive CredentialStatus = "active"
	// CredentialStatusRevoked means the credential was revoked and must be rejected immediately.
	CredentialStatusRevoked CredentialStatus = "revoked"
)

// ValidCredentialScopes is the set of valid CredentialScope values.
var ValidCredentialScopes = map[CredentialScope]bool{
	CredentialScopeAssetsRead:       true,
	CredentialScopeCollectionsWrite: true,
	CredentialScopeDownloadsWrite:   true,
}
