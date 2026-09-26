package errors

import (
	stderrors "errors"

	"github.com/assethub/assethub/internal/repository"
)

// IsNotFound reports whether err is the repository not-found sentinel or wraps it.
func IsNotFound(err error) bool {
	return stderrors.Is(err, repository.ErrNotFound)
}

// AsBusinessError reports whether err is (or wraps) a BusinessError and assigns
// it to target when so.
func AsBusinessError(err error, target **BusinessError) bool {
	return stderrors.As(err, target)
}
