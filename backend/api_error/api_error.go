package api_error

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ApiError struct {
	Message string
	Code    int
	Err     error
}

func (e *ApiError) Error() string {
	return e.Message
}

func NewApiError(message string, code int, err error) *ApiError {
	return &ApiError{Message: message, Code: code, Err: err}
}

func NewInternalServerError(message string, err error) *ApiError {
	return NewApiError(message, http.StatusInternalServerError, err)
}

func NewBadRequestError(message string, err error) *ApiError {
	return NewApiError(message, http.StatusBadRequest, err)
}

func Errorhandler() gin.HandlerFunc {

	return func(c *gin.Context) {
		// we call .Next() to allow middlewares and handlers to run first before checking for errors.
		c.Next()

		if len(c.Errors) > 0 {
			apiError := c.Errors.Last().Err
			switch err := apiError.(type) {
			case *ApiError:
				if err.Err != nil {
					log.Printf("Error: %v", err.Err)
				}
				c.JSON(err.Code, gin.H{"error": err.Message})
			default:
				log.Printf("Unexpected error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			}
			c.Abort()
		}
	}
}
