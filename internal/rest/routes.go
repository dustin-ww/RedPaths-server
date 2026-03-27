package rest

import (
	"RedPaths-server/internal/rest/handlers"
	"RedPaths-server/internal/rest/middleware"
	rpservice "RedPaths-server/pkg/service"
	"RedPaths-server/pkg/service/active_directory"
	"RedPaths-server/pkg/service/change"
	"RedPaths-server/pkg/service/engine"
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
	gpoService *active_directory.GPOService,
	capabilityService *engine.CapabilityService,
	changeService *change.ChangeService,
) {

	projectHandler := handlers.NewProjectHandler(projectService)
	logHandler := handlers.NewLogHandler(logService)
	domainHandler := handlers.NewDomainHandler(projectService, dirNodeService, domainService, gpoService)
	hostHandler := handlers.NewHostHandler(hostService, capabilityService)
	serviceHandler := handlers.NewServiceHandler(serviceService)
	userHandler := handlers.NewUserHandler(userService)
	dirNodeHandler := handlers.NewDirectoryNodeHandler(dirNodeService)
	adHandler := handlers.NewActiveDirectoryHandler(activeDirectoryService)
	capabilityHandler := handlers.NewCapabilityHandler(capabilityService)
	changeHandler := handlers.NewChangeHandler(changeService)

	router.Use(middleware.StripDgraphPrefixMiddleware)
	router.Use(middleware.StripDgraphPrefixMiddleware)
	projects := router.Group("/projects")
	{
		projects.GET("", projectHandler.GetProjectOverviews)
		projects.POST("", projectHandler.CreateProject)
		projects.DELETE("/:projectUID", projectHandler.Delete)

		project := projects.Group("/:projectUID")
		project.Use(middleware.ProjectContext(projectService))
		{
			project.GET("", projectHandler.Get)
			project.PATCH("", middleware.AddPrefixMiddleware("project"), projectHandler.UpdateProject)

			catalog := project.Group("/catalog")
			{
				catalog.GET("/active-directories", projectHandler.GetProjectActiveDirectories)
				// catalog.DELETE("/active-directories/:adUID", adHandler.DeleteActiveDirectory)

				catalog.GET("/domains", projectHandler.GetCatalogDomains)
				catalog.GET("/domains/orphaned", projectHandler.GetOrphanedDomains)
				// catalog.DELETE("/domains/:domainUID", projectHandler.DeleteDomain)

				catalog.GET("/hosts", projectHandler.GetCatalogHosts)
				catalog.GET("/hosts/orphaned", projectHandler.GetOrphanedHosts)
				// catalog.DELETE("/hosts/:hostUID", hostHandler.DeleteHost)

				catalog.GET("/users", projectHandler.GetCatalogUsers)
				catalog.GET("/users/orphaned", projectHandler.GetOrphanedUsers)
				// catalog.DELETE("/users/:userUID", userHandler.DeleteUser)

				catalog.GET("/services", projectHandler.GetCatalogServices)
				//catalog.GET("/services/orphaned", projectHandler.GetOrphanedServices)
				// catalog.DELETE("/services/:serviceUID", serviceHandler.DeleteService)

				catalog.GET("/directory-nodes", projectHandler.GetCatalogDirectoryNodes)
				//catalog.GET("/directory-nodes/orphaned", projectHandler.GetOrphanedDirectoryNodes)
				// catalog.DELETE("/directory-nodes/:dirNodeUID", dirNodeHandler.DeleteDirectoryNode)

				catalog.GET("/capabilities", capabilityHandler.GetCatalogCapabilities)
				// catalog.DELETE("/capabilities/:capabilityUID", capabilityHandler.DeleteCapability)
			}

			// =========================================================
			// ACTIVE DIRECTORIES
			// =========================================================
			project.GET("/active-directories", projectHandler.GetProjectActiveDirectories)
			project.POST("/active-directories", projectHandler.AddProjectActiveDirectory)
			project.GET("/active-directories/:adUID", adHandler.GetProjectActiveDirectory)
			project.PATCH("/active-directories/:adUID", adHandler.UpdateProjectActiveDirectory)
			// project.DELETE("/active-directories/:adUID", adHandler.DeleteActiveDirectory)

			project.GET("/active-directories/:adUID/domains", adHandler.GetActiveDirectoryDomains)
			project.POST("/active-directories/:adUID/domains", adHandler.AddActiveDirectoryDomain)

			// =========================================================
			// DOMAINS – hierarchischer Einstieg
			// =========================================================
			project.GET("/domains/", projectHandler.GetPlacedDomains) // mit AD-Kontext
			project.GET("/hosts", projectHandler.GetPlacedHosts)      // mit Domain/OU-Kontext
			//project.GET("/users", projectHandler.GetPlacedUsers)             // mit OU-Kontext
			project.GET("/directory-nodes", projectHandler.GetDirectoryNodes) // mit Domain-Kontext
			project.GET("/services", projectHandler.GetPlacedServices)
			project.POST("/domains", adHandler.AddActiveDirectoryDomain)
			project.PATCH("/domains/:domainUID", domainHandler.UpdateDomain)
			// project.DELETE("/domains/:domainUID", domainHandler.DeleteDomain)

			project.GET("/domains/:domainUID/changes", handlers.EntityType("Domain"), changeHandler.GetChanges)

			project.GET("/domains/:domainUID/hosts", domainHandler.GetDomainHosts) // nur direkt angehängte Hosts
			project.POST("/domains/:domainUID/hosts", domainHandler.AddDomainHost)
			// project.DELETE("/domains/:domainUID/hosts/:hostUID", domainHandler.RemoveDomainHost)

			// TODO
			//project.GET("/domains/:domainUID/users", domainHandler.GetDomainUsers)          // nur direkt angehängte User
			//project.POST("/domains/:domainUID/users", domainHandler.AddDomainUser)
			// project.DELETE("/domains/:domainUID/users/:userUID", domainHandler.RemoveDomainUser)

			project.GET("/domains/:domainUID/directory-nodes", domainHandler.GetDomainDirectoryNodes)        // direkte Kind-OUs
			project.GET("/domains/:domainUID/directory-nodes/all", domainHandler.GetDeepChildDirectoryNodes) // rekursiv alle
			project.POST("/domains/:domainUID/directory-nodes", domainHandler.AddDomainDirectoryNode)
			// project.DELETE("/domains/:domainUID/directory-nodes/:dirNodeUID", domainHandler.RemoveDirectoryNode)

			project.GET("/domains/:domainUID/gpos", domainHandler.GetDomainGPOs)
			project.GET("/domains/:domainUID/gpos/all", domainHandler.GetDomainGPOLib)
			project.POST("/domains/:domainUID/gpos", domainHandler.LinkDomainGPO)
			// project.DELETE("/domains/:domainUID/gpos/:gpoUID", domainHandler.UnlinkDomainGPO)

			// =========================================================
			// DIRECTORY NODES (OU / Container)
			// =========================================================
			project.PATCH("/directory-nodes/:dirNodeUID", dirNodeHandler.UpdateDirectoryNode)
			// project.DELETE("/directory-nodes/:dirNodeUID", dirNodeHandler.DeleteDirectoryNode)

			// project.DELETE("/directory-nodes/:dirNodeUID/hosts/:hostUID", dirNodeHandler.RemoveDirectoryNodeHost)

			project.GET("/directory-nodes/:dirNodeUID/users", dirNodeHandler.GetDirectoryNodeUsers) // direkte User dieser OU
			//project.POST("/directory-nodes/:dirNodeUID/users", dirNodeHandler.AddDirectoryNodeUser)
			// project.DELETE("/directory-nodes/:dirNodeUID/users/:userUID", dirNodeHandler.RemoveDirectoryNodeUser)

			project.GET("/directory-nodes/:dirNodeUID/children", dirNodeHandler.GetChildDirectoryNodes)         // direkte Kind-OUs
			project.GET("/directory-nodes/:dirNodeUID/children/all", dirNodeHandler.GetDeepChildDirectoryNodes) // rekursiv alle
			project.POST("/directory-nodes/:dirNodeUID/children", dirNodeHandler.AddChildDirectoryNode)
			// project.DELETE("/directory-nodes/:dirNodeUID/children/:childUID", dirNodeHandler.RemoveChildDirectoryNode)

			// project.GET("/directory-nodes/:dirNodeUID/acls", dirNodeHandler.GetACLs) // TODO implement

			// =========================================================
			// HOSTS
			// =========================================================
			project.POST("/hosts", hostHandler.CreateHost)
			project.PATCH("/hosts/:hostUID", hostHandler.UpdateHost)
			// project.DELETE("/hosts/:hostUID", hostHandler.DeleteHost)

			project.GET("/hosts/:hostUID/changes", handlers.EntityType("Host"), changeHandler.GetChanges)

			project.GET("/hosts/:hostUID/services", serviceHandler.GetServices)
			project.POST("/hosts/:hostUID/services", hostHandler.AddService)
			// project.DELETE("/hosts/:hostUID/services/:serviceUID", hostHandler.RemoveService)

			project.GET("/hosts/:hostUID/capabilities", hostHandler.GetLinkedCapabilities)
			project.POST("/hosts/:hostUID/capabilities", hostHandler.AddCapability)
			// project.DELETE("/hosts/:hostUID/capabilities/:capabilityUID", hostHandler.RemoveCapability)

			// =========================================================
			// SERVICES
			// =========================================================
			project.PATCH("/services/:serviceUID", serviceHandler.UpdateService)
			// project.DELETE("/services/:serviceUID", serviceHandler.DeleteService)

			// =========================================================
			// USERS
			// =========================================================
			project.POST("/users", userHandler.CreateUser)
			project.PATCH("/users/:userUID", userHandler.UpdateUser)
			// project.DELETE("/users/:userUID", userHandler.DeleteUser)

			project.GET("/users/:userUID/changes", handlers.EntityType("User"), changeHandler.GetChanges)

			// =========================================================
			// CAPABILITIES
			// =========================================================
			// project.DELETE("/capabilities/:capabilityUID", capabilityHandler.DeleteCapability)

			// =========================================================
			// TARGETS
			// =========================================================
			project.GET("/targets", projectHandler.GetTargets)
			project.POST("/targets", middleware.AddPrefixMiddleware("project"), projectHandler.CreateTarget)
			// project.DELETE("/targets/:targetUID", projectHandler.DeleteTarget)

			// =========================================================
			// LOGS
			// =========================================================
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
