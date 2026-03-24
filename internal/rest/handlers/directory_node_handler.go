package handlers

import (
	rpad "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/service/active_directory"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DirectoryNodeHandler struct {
	directoryNodeService *active_directory.DirectoryNodeService
}

func NewDirectoryNodeHandler(directoryNodeService *active_directory.DirectoryNodeService) *DirectoryNodeHandler {
	return &DirectoryNodeHandler{
		directoryNodeService: directoryNodeService,
	}
}

func (h *DirectoryNodeHandler) AddUser(c *gin.Context) {
	uid := c.Param("directoryNodeUID")

	type AddUserRequest struct {
		Name string `json:"name" binding:"required" validate:"required"`
	}

	var request AddUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	user := &rpad.User{
		SAMAccountName: request.Name,
	}

	securityPrincipal, err := h.directoryNodeService.AddSecurityPrincipal(c.Request.Context(), uid, user, "UserInput")
	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to add security principal",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                  "Security Principal has been added successfully",
		"added_security_principal": securityPrincipal,
	})
}

/*
	func (h *DirectoryNodeHandler) GetACL(c *gin.Context) {
		uid := c.Param("directoryNodeUID")
		if uid == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Directory Node UID is required",
			})
			return
		}

		securityPrincipals, err := h.directoryNodeService.(
			c.Request.Context(),
			uid,
		)

		if err != nil {
			errReturn := gin.H{
				"error":   "Failed to retrieve security principals for provided directory node",
				"details": err.Error(),
			}
			log.Println(err)
			c.JSON(http.StatusInternalServerError, errReturn)
		}

		c.JSON(http.StatusOK, securityPrincipals)
	}
*/
func (h *DirectoryNodeHandler) GetDirectoryNodeUsers(c *gin.Context) {
	uid := c.Param("directoryNodeUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Directory Node UID is required",
		})
		return
	}

	securityPrincipals, err := h.directoryNodeService.GetDirectoryNodeSecurityPrincipals(
		c.Request.Context(),
		uid,
	)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve security principals for provided directory node",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
	}

	c.JSON(http.StatusOK, securityPrincipals)
}

/*func (h *DirectoryNodeHandler) GetProjectActiveDirectory(c *gin.Context) {
	uid := c.Param("domainUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Active Directory UID is required",
		})
		return
	}

	project, err := h.activeDirectoryService.GetProjectActiveDirectory(
		c.Request.Context(),
		uid,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve active directory",
			"details": err.Error(),
		})
		return
	}
	log.Println(*project)
	c.JSON(http.StatusOK, project)
}
*/

func (h *DirectoryNodeHandler) AddChildDirectoryNode(c *gin.Context) {
	type AddChildDirectoryNodeRequest struct {
		Name string `json:"name" binding:"required" validate:"required"`
	}

	var request AddChildDirectoryNodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to add a new directory node",
			"details": err.Error(),
		})
		return
	}

	dirNodeUID := c.Param("dirNodeUID")

	directoryNode := &rpad.DirectoryNode{
		Name: request.Name,
	}

	createdDirectoryNode, err := h.directoryNodeService.AddChildDirectoryNode(
		c.Request.Context(),
		dirNodeUID,
		directoryNode,
		"user",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add directory node into parent directory node",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":               "success",
		"message":              "New child directory node has been added to the directory node",
		"added_directory_node": createdDirectoryNode,
	})

}

func (h *DirectoryNodeHandler) GetChildDirectoryNodes(c *gin.Context) {

	dirNodeUID := c.Param("dirNodeUID")

	childDirectoryNodes, err := h.directoryNodeService.GetAllChildDirectoryNodes(
		c.Request.Context(),
		dirNodeUID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get child directory nodes",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, childDirectoryNodes)

}

func (h *DirectoryNodeHandler) GetDeepChildDirectoryNodes(c *gin.Context) {

	dirNodeUID := c.Param("dirNodeUID")

	childDirectoryNodes, err := h.directoryNodeService.GetAllChildrenRecursive(
		c.Request.Context(),
		dirNodeUID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get child directory nodes",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, childDirectoryNodes)

}

func (h *DirectoryNodeHandler) UpdateDirectoryNode(c *gin.Context) {
	//project := restcontext.Project(c)
	uid := c.Param("adUID")
	var fieldsToUpdate map[string]interface{}

	if err := c.BindJSON(&fieldsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
		return
	}

	updatedDirectoryNode, err := h.directoryNodeService.UpdateDirectoryNode(c.Request.Context(), uid, "UserInput", fieldsToUpdate)

	if err != nil {
		log.Printf("Sending 500 response while updating directory node because: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update directory node",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Directory Node updated successfully",
		"updated_project": updatedDirectoryNode,
	})
}
