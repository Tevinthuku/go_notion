package router

import (
	"go_notion/backend/api_error"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	// In release mode, Gin typically disables some debug features and optimizes for performance,
	// which is suitable for production environments
	if env := os.Getenv("GIN_MODE"); env == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.Default()
	router.Use(api_error.Errorhandler())
	router.Use(IPRateLimiter(RateLimitConfig{Requests: 60, Period: time.Minute, Burst: 5}))
	return router
}
