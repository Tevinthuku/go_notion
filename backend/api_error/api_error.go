package api_error

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ApiError struct {
	// Message contains the user-facing error description
	Message string
	// Code contains the HTTP status code
	Code int
	// Err contains the underlying error for logging purposes
	Err error
}

func (e *ApiError) Error() string {
	return e.Message
}

// NewInternalServerError creates a new API error with StatusInternalServerError
func NewInternalServerError(message string, err error) *ApiError {
	return newApiError(message, http.StatusInternalServerError, err)
}

// NewBadRequestError creates a new API error with StatusBadRequest
func NewBadRequestError(message string, err error) *ApiError {
	return newApiError(message, http.StatusBadRequest, err)
}

// NewUnauthorizedError creates a new API error with StatusUnauthorized
func NewUnauthorizedError(message string, err error) *ApiError {
	return newApiError(message, http.StatusUnauthorized, err)
}

func newApiError(message string, code int, err error) *ApiError {
	return &ApiError{Message: message, Code: code, Err: err}
}

func Errorhandler() gin.HandlerFunc {

	return func(c *gin.Context) {
		// we call .Next() to allow middlewares and handlers to run first before checking for errors.
		c.Next()
		if len(c.Errors) > 0 {
			var errs []gin.H
			for _, err := range c.Errors {
				switch err := err.Err.(type) {
				case *ApiError:
					log.Printf("ApiError: %v", err.Err)
					errs = append(errs, gin.H{"error": err.Message})
				default:
					log.Printf("Unexpected error: %v", err)
					errs = append(errs, gin.H{"error": "Internal server error"})
				}
			}
			// Return all errors with the status code of the most severe error
			c.JSON(highestStatusCode(c.Errors), gin.H{"errors": errs})
			c.Abort()
		}
	}
}

func highestStatusCode(errs []*gin.Error) int {
	highest := http.StatusOK
	for _, err := range errs {
		if apiErr, ok := err.Err.(*ApiError); ok && apiErr.Code > highest {
			highest = apiErr.Code
		}
	}
	// if no error code is set, return 500. We expect the status code to not be 200 since we have caught at least 1 error.
	if highest == http.StatusOK {
		return http.StatusInternalServerError
	}
	return highest
}
