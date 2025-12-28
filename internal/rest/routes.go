package rest

import (
	"RedPaths-server/internal/rest/handlers"
	"RedPaths-server/internal/rest/middleware"
	rpservice "RedPaths-server/pkg/service"
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

func RegisterProjectHandlers(
	router *gin.Engine,
	projectService *active_directory.ProjectService,
	logService *rpservice.LogService,
	domainService *active_directory.DomainService,
	hostService *active_directory.HostService,
	serviceService *active_directory.ServiceService,
	userService *active_directory.UserService) {

	projectHandler := handlers.NewProjectHandler(projectService)
	logHandler := handlers.NewLogHandler(logService)
	domainHandler := handlers.NewDomainHandler(projectService, domainService)
	hostHandler := handlers.NewHostHandler(hostService)
	serviceHandler := handlers.NewServiceHandler(serviceService)
	userHandler := handlers.NewUserHandler(userService)

	projects := router.Group("/projects")
	{
		projects.GET("/overviews", projectHandler.GetProjectOverviews)
		projects.DELETE("/:projectUID", projectHandler.Delete)
		projects.GET("", projectHandler.GetProjectOverviews)
		projects.POST("", projectHandler.CreateProject)

		project := projects.Group("/:projectUID")
		project.Use(middleware.ProjectContext(projectService))
		{
			project.GET("", projectHandler.Get)
			project.PATCH("", projectHandler.UpdateProject)

			domains := project.Group("/domains")
			{
				domains.GET("", projectHandler.GetDomains)
				domains.POST("", projectHandler.AddDomain)
				domain := domains.Group("/:domainUID")
				domain.Use(middleware.DomainContext(projectService))
				{
					domains.PATCH("", domainHandler.UpdateDomain)

					domain.GET("/hosts", domainHandler.GetHosts)
					domain.POST("/hosts", domainHandler.AddHost)
				}

			}

			hosts := project.Group("/hosts")
			{
				hosts.POST("", hostHandler.CreateHost)
				hosts.GET("", projectHandler.GetHosts)

				host := hosts.Group("/:hostUID")
				host.Use(middleware.HostContext(projectService))
				{
					host.PATCH("", hostHandler.UpdateHost)

					host.GET("/services", serviceHandler.GetServices)

					service := host.Group("/:serviceUID")
					service.Use(middleware.ServiceContext(hostService))
					{
						service.PATCH("", serviceHandler.UpdateService)
					}

				}
			}

			users := project.Group("/users")
			{
				users.GET("", projectHandler.GetUsers)
				users.POST("", userHandler.CreateUser)

				user := users.Group("/:userUID")
				user.Use(middleware.UserContext(projectService))
				{
					user.PATCH("", userHandler.UpdateUser)
				}

			}

			targets := project.Group("/targets")
			{
				targets.GET("", projectHandler.GetTargets)
				targets.POST("", projectHandler.CreateTarget)
			}

			logs := project.Group("/logs")
			{
				logs.GET("", logHandler.GetLogs)
				logs.POST("/query", logHandler.GetLogsWithOptions)
				logs.GET("/types", logHandler.GetLogTypes)
				logs.GET("/mkeys", logHandler.GetModuleKeySet)
			}
		}
	}
}

func RegisterRedPathsModuleHandlers(router *gin.Engine, redPathsModuleService *redpaths.ModuleService, projectService *active_directory.ProjectService) {
	moduleHandler := handlers.NewRedPathsModuleHandler(redPathsModuleService)

	redPaths := router.Group("/redpaths")
	{

		modules := redPaths.Group("/modules")
		{
			modules.GET("", moduleHandler.GetModules)
			modules.GET("/graph", moduleHandler.GetModuleInheritanceGraph)

			module := modules.Group("/:moduleKey")
			module.Use(middleware.ModuleContext(redPathsModuleService))
			{
				module.GET("/run", moduleHandler.RunModule)
				module.GET("/options", moduleHandler.GetModuleOptions)

				moduleVector := module.Group("/vector")
				{
					moduleVector.POST("/run", moduleHandler.RunAttackVector)
					moduleVector.GET("/options", moduleHandler.GetAttackVectorOptions)
				}
			}

		}
		project := redPaths.Group("/:projectUID")
		project.Use(middleware.ProjectContext(projectService))
		{
			project.GET("/vruns", moduleHandler.GetVectorRuns)
			project.GET("/mruns", moduleHandler.GetModuleRuns)
		}
	}
}
