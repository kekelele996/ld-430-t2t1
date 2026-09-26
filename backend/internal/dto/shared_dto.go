package dto

// SharedAssetQuery are the read-only asset filter parameters exposed to external services.
// Status is intentionally absent: shared callers only ever see published assets.
type SharedAssetQuery struct {
	Keyword     string `form:"keyword"`
	FileType    string `form:"file_type"`
	LicenseType string `form:"license_type"`
	Tag         string `form:"tag"`
	CategoryID  string `form:"category_id"`
	Sort        string `form:"sort"`
	Page        string `form:"page"`
	PageSize    string `form:"page_size"`
}

// SharedCollectionRequest creates or updates a collection on behalf of the team.
type SharedCollectionRequest struct {
	Name          string `json:"name" validate:"required,min=1,max=100"`
	Description   string `json:"description"`
	CoverImageURL string `json:"cover_image_url"`
	IsPublic      bool   `json:"is_public"`
}

// SharedDownloadRequest records a download on behalf of the team.
type SharedDownloadRequest struct {
	Purpose        string `json:"purpose" validate:"required,oneof=Personal Commercial Editorial"`
	LicenseVersion string `json:"license_version"`
}
