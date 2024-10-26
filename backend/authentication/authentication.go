package authentication

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthenticationContext struct {
	db *pgxpool.Pool
}

func NewAuthenticationContext(db *pgxpool.Pool) *AuthenticationContext {
	return &AuthenticationContext{db: db}
}

func (ac *AuthenticationContext) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/sign-up", ac.signUp)
}

type SignUpInput struct {
	Email    string `json:"email" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (ac *AuthenticationContext) signUp(c *gin.Context) {
	var input SignUpInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
}
