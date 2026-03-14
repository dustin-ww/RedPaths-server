package handlers

import (
	restcontext "RedPaths-server/internal/rest/context"
	"RedPaths-server/internal/rest/requests"
	"RedPaths-server/pkg/model"
	rpad "RedPaths-server/pkg/model/active_directory"
	gpo2 "RedPaths-server/pkg/model/active_directory/gpo"
	"RedPaths-server/pkg/service/active_directory"

	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type DomainHandler struct {
	projectService       *active_directory.ProjectService
	domainService        *active_directory.DomainService
	gpoService           *active_directory.GPOService
	directoryNodeService *active_directory.DirectoryNodeService
}

func NewDomainHandler(projectService *active_directory.ProjectService, directoryNodeService *active_directory.DirectoryNodeService, domainService *active_directory.DomainService, gpoService *active_directory.GPOService) *DomainHandler {
	return &DomainHandler{
		projectService:       projectService,
		domainService:        domainService,
		gpoService:           gpoService,
		directoryNodeService: directoryNodeService,
	}
}

/*
func (h *DomainHandler) GetHosts(c *gin.Context) {
	uid := c.Param("domainUID")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Domain UID is required",
		})
		return
	}

	hosts, err := h.domainService.GetDomainHosts(
		c.Request.Context(),
		uid,
	)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve hosts for given domain uid",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
	}

	c.JSON(http.StatusOK, hosts)
}*/

func (h *DomainHandler) GetHosts(c *gin.Context) {
	//domain := restcontext.Domain(c)
	domainUID := c.Param("domainUID")

	hosts, err := h.domainService.GetDomainHosts(
		c.Request.Context(),
		domainUID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, hosts)
}

func (h *DomainHandler) GetGPOs(c *gin.Context) {
	//domain := restcontext.Domain(c)
	domainUID := c.Param("domainUID")

	hosts, err := h.domainService.GetDomainHosts(
		c.Request.Context(),
		domainUID,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, hosts)
}

func (h *DomainHandler) GetDirectoryNodes(c *gin.Context) {
	domainUID := c.Param("domainUID")

	directoryNodes, err := h.domainService.GetDomainDirectoryNodes(
		c.Request.Context(),
		domainUID)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to retrieve directory nodes for given domain uid",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
	}
	c.JSON(http.StatusOK, directoryNodes)
}

func (h *DomainHandler) GetDeepChildDirectoryNodes(c *gin.Context) {
	domainUID := c.Param("domainUID")

	directoryNodes, err := h.directoryNodeService.GetAllDirectoryNodesInDomain(
		c.Request.Context(),
		domainUID)

	if err != nil {
		errReturn := gin.H{
			"error":   "Failed to recursive retrieve all directory nodes for given domain uid",
			"details": err.Error(),
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, errReturn)
	}
	c.JSON(http.StatusOK, directoryNodes)

}

func (h *DomainHandler) AddDirectoryNode(c *gin.Context) {

	var request requests.AddDirectoryNodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to add a new directory node",
			"details": err.Error(),
		})
		return
	}

	domainUID := c.Param("domainUID")

	directoryNode := &rpad.DirectoryNode{
		Name: request.Name,
	}

	createdDirectoryNode, err := h.domainService.AddDirectoryNode(
		c.Request.Context(),
		request.AssertionContext,
		domainUID,
		directoryNode,
		"user",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add directory node into domain",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":               "success",
		"message":              "New directory node has been added to the domain",
		"added_directory_node": createdDirectoryNode,
	})

}

func (h *DomainHandler) AddGPOLink(c *gin.Context) {
	type AddDirectoryNodeRequest struct {
		Name      string `json:"name" binding:"required" validate:"required"`
		LinkOrder int    `json:"linkOrder" binding:"required" validate:"required"`
	}

	var request AddDirectoryNodeRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request to add a new gpo link",
			"details": err.Error(),
		})
		return
	}

	domainUID := c.Param("domainUID")

	// INIT
	gpo := gpo2.GPO{Name: request.Name}
	gpoLink := gpo2.Link{LinkOrder: request.LinkOrder}
	gpoLink.LinksTo = &gpo

	createdGPOLink, err := h.gpoService.LinkGPOToContainer(
		c.Request.Context(),
		domainUID,
		"Domain",
		&gpoLink,
		"user",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add gpo link to domain",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":               "success",
		"message":              "New gpo link has been added to the domain",
		"added_directory_node": createdGPOLink,
	})

}

func (h *DomainHandler) UpdateDomain(c *gin.Context) {

	domain := restcontext.Domain(c)
	var fieldsToUpdate map[string]interface{}

	if err := c.BindJSON(&fieldsToUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_JSON"})
		return
	}
	updatedDomain, err := h.domainService.UpdateDomain(c.Request.Context(), domain.UID, "UserInput", fieldsToUpdate)

	if err != nil {
		log.Printf("Sending 500 response while updating domain because: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to update domain",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Domain updated successfully",
		"updated_domain": updatedDomain,
	})
}

func (h *DomainHandler) AddHost(c *gin.Context) {
	//domain := restcontext.Domain(c)
	domainUID := c.Param("domainUID")

	var request requests.AddHostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data to create a new host",
			"details": err.Error(),
		})
		return
	}

	host := &model.Host{
		IP: request.Ip,
	}

	result, err := h.domainService.AddHost(
		c.Request.Context(),
		request.AssertionContext,
		domainUID,
		host,
		"user",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add host",
			"details": err.Error(),
		})
		log.Printf("Sending client 500 error response for adding host to domain %s with message %s", domainUID, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "New user has been created",
		"created": result,
	})
}

func (h *DomainHandler) CreateUncertainDomain(c *gin.Context) {
	//domain := restcontext.Domain(c)
	domainUID := c.Param("domainUID")

	var request requests.AddHostRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request data to create a new host",
			"details": err.Error(),
		})
		return
	}

	host := &model.Host{
		IP: request.Ip,
	}

	result, err := h.domainService.AddHost(
		c.Request.Context(),
		request.AssertionContext,
		domainUID,
		host,
		"user",
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add host",
			"details": err.Error(),
		})
		log.Printf("Sending client 500 error response for adding host to domain %s with message %s", domainUID, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "New user has been created",
		"created": result,
	})
}
