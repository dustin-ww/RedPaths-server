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
	userService *active_directory.UserService,
	dirNodeService *active_directory.DirectoryNodeService,
	activeDirectoryService *active_directory.ActiveDirectoryService,
) {

	projectHandler := handlers.NewProjectHandler(projectService)
	logHandler := handlers.NewLogHandler(logService)
	domainHandler := handlers.NewDomainHandler(projectService, domainService)
	hostHandler := handlers.NewHostHandler(hostService)
	serviceHandler := handlers.NewServiceHandler(serviceService)
	userHandler := handlers.NewUserHandler(userService)
	dirNodeHandler := handlers.NewDirectoryNodeHandler(dirNodeService)
	adHandler := handlers.NewActiveDirectoryHandler(activeDirectoryService)

	projects := router.Group("/projects")
	{
		projects.GET("", projectHandler.GetProjectOverviews)
		projects.POST("", projectHandler.CreateProject)
		projects.DELETE("/:projectUID", projectHandler.Delete)

		project := projects.Group("/:projectUID")
		project.Use(middleware.ProjectContext(projectService))
		{
			project.GET("", projectHandler.Get)
			project.PATCH("", projectHandler.UpdateProject)

			// --- Active Directories ---
			project.GET("/active-directories", projectHandler.GetActiveDirectories)
			project.POST("/active-directories", projectHandler.AddActiveDirectory)
			project.GET("/active-directories/:adUID", adHandler.Get)
			project.PATCH("/active-directories/:adUID", adHandler.UpdateActiveDirectory)
			project.PATCH("/active-directories/:adUID/domains", adHandler.GetDomains)
			project.POST("/active-directories/:adUID/domains", adHandler.AddDomain)

			// --- Domains ---
			//project.GET("/domains", adHandler.GetDomains) TODO implement
			//project.POST("/domains", adHandler.AddDomain) TODO implement
			project.PATCH("/domains/:domainUID", domainHandler.UpdateDomain)
			project.GET("/domains/:domainUID/hosts", domainHandler.GetHosts)
			project.POST("/domains/:domainUID/hosts", domainHandler.AddHost)
			project.GET("/domains/:domainUID/directory-nodes", domainHandler.GetDirectoryNodes)
			project.POST("/domains/:domainUID/directory-nodes", domainHandler.AddDirectoryNode)
			//project.POST("/domains/:domainUID/users", domainHandler.)

			// --- Directory Nodes (OU / Container) ---
			//project.GET("/directory-nodes", dirNodeHandler.GetDirectoryNodes) TODO implement
			//project.POST("/directory-nodes", dirNodeHandler.CreateDirectoryNode) TODO implement
			/*project.GET("/directory-nodes/:dirNodeUID", dirNodeHandler.GetDirectoryNode)*/
			project.PATCH("/directory-nodes/:dirNodeUID", dirNodeHandler.UpdateDirectoryNode)
			project.GET("/directory-nodes/:dirNodeUID/users", dirNodeHandler.GetUsers)

			// --- Hosts ---
			project.GET("/hosts", projectHandler.GetHosts)
			project.POST("/hosts", hostHandler.CreateHost)
			project.PATCH("/hosts/:hostUID", hostHandler.UpdateHost)
			project.GET("/hosts/:hostUID/services", serviceHandler.GetServices)
			project.PATCH("/services/:serviceUID", serviceHandler.UpdateService)

			// --- Users ---
			project.GET("/users", projectHandler.GetUsers)
			project.POST("/users", userHandler.CreateUser)
			project.PATCH("/users/:userUID", userHandler.UpdateUser)

			// --- Targets ---
			project.GET("/targets", projectHandler.GetTargets)
			project.POST("/targets", projectHandler.CreateTarget)

			// --- Logs ---
			project.GET("/logs", logHandler.GetLogs)
			project.POST("/logs/query", logHandler.GetLogsWithOptions)
			project.GET("/logs/types", logHandler.GetLogTypes)
			project.GET("/logs/mkeys", logHandler.GetModuleKeySet)
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
