package handlers

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/service/active_directory"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *active_directory.UserService
}

func NewUserHandler(userService *active_directory.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// CreateUser Function to create a user without domain
func (h *UserHandler) CreateUser(c *gin.Context) {
	type CreateUserRequest struct {
		Name string `json:"name" binding:"required"`
	}

	var request CreateUserRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to create a new user",
			"details": err.Error(),
		})
		return
	}

	user := &model.ADUser{
		Name: request.Name,
	}

	projectUid := c.Param("projectUID")
	createdUser, err := h.userService.Create(
		c.Request.Context(),
		user,
		projectUid,
		"UserInput",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create a new user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":       "success",
		"message":      "New user has been created",
		"created_user": createdUser,
	})
}
