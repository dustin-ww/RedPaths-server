package context

import (
	"RedPaths-server/pkg/model"
	"RedPaths-server/pkg/model/redpaths"

	"github.com/gin-gonic/gin"
)

func Project(c *gin.Context) *model.Project {
	return c.MustGet("project").(*model.Project)
}

func Domain(c *gin.Context) *model.Domain {
	return c.MustGet("domain").(*model.Domain)
}

func Module(c *gin.Context) *redpaths.Module {
	return c.MustGet("module").(*redpaths.Module)
}

func Host(c *gin.Context) *model.Host {
	return c.MustGet("host").(*model.Host)
}

func Service(c *gin.Context) *model.Service {
	return c.MustGet("service").(*model.Service)
}

func User(c *gin.Context) *model.ADUser {
	return c.MustGet("user").(*model.ADUser)
}
