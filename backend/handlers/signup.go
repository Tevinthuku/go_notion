package handlers

import (
	"context"
	"fmt"
	"go_notion/backend/api_error"
	"go_notion/backend/auth"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SignUpHandler struct {
	db             *pgxpool.Pool
	tokenGenerator auth.TokenGenerator
}

func NewSignUpHandler(db *pgxpool.Pool, tokenGenerator auth.TokenGenerator) (*SignUpHandler, error) {
	if db == nil || tokenGenerator == nil {
		return nil, fmt.Errorf("db and tokenGenerator cannot be nil")
	}
	return &SignUpHandler{db, tokenGenerator}, nil
}

type SignUpInput struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=5"`
}

func (s *SignUpHandler) SignUp(c *gin.Context) {

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	var input SignUpInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(api_error.NewBadRequestError(err.Error(), err))
		return
	}

	hashedPassword, hashErr := auth.HashPassword(input.Password)
	if hashErr != nil {
		if hashErr.IsPasswordValidationError() {
			c.Error(api_error.NewBadRequestError(hashErr.Error(), hashErr))
		} else {
			c.Error(api_error.NewInternalServerError("failed to process password", hashErr))
		}
		return
	}

	input.Password = string(hashedPassword)

	// the check and insert should be in a transaction to prevent race conditions in case of concurrent requests
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		c.Error(api_error.NewInternalServerError("internal server error", err))
		return
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			if cerr := c.Error(api_error.NewInternalServerError("internal server error", err)); cerr != nil {
				log.Printf("failed to rollback transaction: %v", cerr)
			}
		}
	}()

	var existingEmailCount, existingUserNameCount int
	err = tx.QueryRow(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE email = $1) as email_count,
			COUNT(*) FILTER (WHERE username = $2) as username_count
		FROM users 
		WHERE email = $1 OR username = $2
	`, input.Email, input.Username).Scan(&existingEmailCount, &existingUserNameCount)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to validate user", err))
		return
	}

	if existingEmailCount > 0 {
		c.Error(api_error.NewBadRequestError("email already in use", nil))
		return
	}

	if existingUserNameCount > 0 {
		c.Error(api_error.NewBadRequestError("username already taken", nil))
		return
	}

	var userID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, username, password) 
		VALUES ($1, $2, $3) 
		RETURNING id
	`, input.Email, input.Username, input.Password).Scan(&userID)

	if err != nil {
		c.Error(api_error.NewInternalServerError("failed to create user.", err))
		return
	}

	if err := tx.Commit(ctx); err != nil {
		c.Error(api_error.NewInternalServerError("internal server error", err))
		return
	}

	token, err := s.tokenGenerator.Generate(userID)
	if err != nil {
		c.Error(api_error.NewInternalServerError("authentication failed", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (s *SignUpHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/auth/signup", s.SignUp)
}
