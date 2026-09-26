package constants

// SharedScope enumerates the authorization scopes grantable to a team shared credential.
type SharedScope string

const (
	// ScopeAssetsRead allows read-only access to published assets.
	ScopeAssetsRead SharedScope = "assets:read"
	// ScopeCollectionsWrite allows maintaining (creating/updating) favorite collections.
	ScopeCollectionsWrite SharedScope = "collections:write"
	// ScopeDownloadsWrite allows registering asset downloads.
	ScopeDownloadsWrite SharedScope = "downloads:write"
)

// ValidSharedScopes is the set of grantable shared credential scopes.
var ValidSharedScopes = map[SharedScope]bool{
	ScopeAssetsRead:       true,
	ScopeCollectionsWrite: true,
	ScopeDownloadsWrite:   true,
}
