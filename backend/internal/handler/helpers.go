package handler

import (
	"github.com/assethub/assethub/internal/errors"
	"github.com/gin-gonic/gin"
)

// respondBusiness attaches a business error to the gin context so the unified
// error handler turns it into a standard response.
func respondBusiness(c *gin.Context, code int, message string) {
	_ = c.Error(errors.NewBusinessError(code, message))
}

// respondValidation attaches a validation error to the gin context.
func respondValidation(c *gin.Context, err error) {
	_ = c.Error(errors.NewValidationError(err))
}
