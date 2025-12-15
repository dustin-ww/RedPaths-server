package rest

import (
	"RedPaths-server/internal/rest/handlers"
	"RedPaths-server/pkg/service"
	"RedPaths-server/pkg/service/active_directory"
	"RedPaths-server/pkg/service/redpaths"

	"github.com/gin-gonic/gin"
)

func RegisterServerHandlers(router *gin.Engine) {
	serverHandler := handlers.NewServerHandler()
	serverGroup := router.Group("/server")
	{
		serverGroup.GET("/health", serverHandler.GetHealth)
	}
}

func RegisterProjectHandlers(router *gin.Engine, projectService *active_directory.ProjectService, logService *service.LogService, domainService *active_directory.DomainService, hostService *active_directory.HostService, serviceService *active_directory.ServiceService) {
	projectHandler := handlers.NewProjectHandler(projectService)
	logHandler := handlers.NewLogHandler(logService)
	domainHandler := handlers.NewDomainHandler(projectService, domainService)
	hostHandler := handlers.NewHostHandler(hostService)
	serviceHandler := handlers.NewServiceHandler(serviceService)

	projectGroup := router.Group("/project")
	{
		projectGroup.GET("/overviews", projectHandler.GetProjectOverviews)
		projectGroup.DELETE("/:projectUID", projectHandler.Delete)
		projectGroup.GET("", projectHandler.GetProjectOverviews)
		projectGroup.POST("", projectHandler.CreateProject)

		projectItemGroup := projectGroup.Group("/:projectUID")
		{
			projectItemGroup.GET("", projectHandler.Get)
			projectItemGroup.PATCH("", projectHandler.UpdateProject)

			domainsGroup := projectItemGroup.Group("/domains")
			{
				domainsGroup.GET("", projectHandler.GetDomains)
				domainsGroup.POST("", projectHandler.AddDomain)

				domainItemGroup := domainsGroup.Group("/:domainUID")
				{
					domainItemGroup.GET("/hosts", domainHandler.GetHosts)
					domainItemGroup.POST("/hosts", domainHandler.AddHost)
				}

			}

			hostsGroup := projectItemGroup.Group("/hosts")
			{
				hostsGroup.POST("", hostHandler.CreateHost)
				hostsGroup.GET("", projectHandler.GetHosts)

				hostItemGroup := hostsGroup.Group("/:hostUID")
				{
					hostItemGroup.GET("/services", serviceHandler.GetServices)
				}
			}

			targetsGroup := projectItemGroup.Group("/targets")
			{
				targetsGroup.GET("", projectHandler.GetTargets)
				targetsGroup.POST("", projectHandler.CreateTarget)
			}

			logsGroup := projectItemGroup.Group("/logs")
			{
				logsGroup.GET("", logHandler.GetLogs)
				logsGroup.POST("/query", logHandler.GetLogsWithOptions)
				logsGroup.GET("/types", logHandler.GetLogTypes)
				logsGroup.GET("/mkeys", logHandler.GetModuleKeySet)
			}
		}
	}
}

func RegisterRedPathsModuleHandlers(router *gin.Engine, redPathsModuleService *redpaths.ModuleService) {
	moduleHandler := handlers.NewRedPathsModuleHandler(redPathsModuleService)

	redPathsGroup := router.Group("/redpaths")
	{

		moduleGroup := redPathsGroup.Group("/modules")
		{
			moduleGroup.GET("", moduleHandler.GetModules)
			moduleGroup.GET("/graph", moduleHandler.GetModuleInheritanceGraph)

			moduleItemGroup := moduleGroup.Group("/:moduleKey")
			{
				moduleItemGroup.GET("/run", moduleHandler.RunModule)
				moduleItemGroup.GET("/options", moduleHandler.GetModuleOptions)

				moduleItemVectorGroup := moduleItemGroup.Group("/vector")
				{
					moduleItemVectorGroup.POST("/run", moduleHandler.RunAttackVector)
					moduleItemVectorGroup.GET("/options", moduleHandler.GetAttackVectorOptions)
				}
			}

		}
		projectItemGroup := redPathsGroup.Group("/:projectUID")
		{
			projectItemGroup.GET("/vruns", moduleHandler.GetVectorRuns)
			projectItemGroup.GET("/mruns", moduleHandler.GetModuleRuns)
		}
	}
}
