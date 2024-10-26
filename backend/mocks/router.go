package mocks

import (
	"go_notion/backend/api_error"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	router := gin.Default()
	router.Use(api_error.Errorhandler())
	return router
}
