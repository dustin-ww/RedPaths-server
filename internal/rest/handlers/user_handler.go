package handlers

import (
	restcontext "RedPaths-server/internal/rest/context"
	active_directory2 "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/service/active_directory"
	"log"
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

	user := &active_directory2.User{
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

func (h *UserHandler) UpdateUser(c *gin.Context) {

	user := restcontext.User(c)
	var fieldsToUpdate map[string]interface{}

	if err := c.BindJSON(&fieldsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
		return
	}
	updatedUser, err := h.userService.UpdateUser(c.Request.Context(), user.UID, "UserInput", fieldsToUpdate)

	if err != nil {
		log.Printf("Sending 500 response while updating user because: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update user",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "User updated successfully",
		"updated_user": updatedUser,
	})
}
