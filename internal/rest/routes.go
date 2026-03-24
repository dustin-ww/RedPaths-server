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
				// --- GET: Lesen aus dem globalen Katalog ---
				catalog.GET("/domains", projectHandler.GetCatalogDomains)
				catalog.GET("/hosts", projectHandler.GetCatalogHosts)
				catalog.GET("/users", projectHandler.GetCatalogUsers)
				catalog.GET("/services", projectHandler.GetCatalogServices)
				catalog.GET("/directory-nodes", projectHandler.GetCatalogDirectoryNodes)
				catalog.GET("/capabilities", capabilityHandler.GetCatalogCapabilities)

				// --- Orphaned: Entities ohne bekannten Parent ---
				catalog.GET("/domains/orphaned", projectHandler.GetOrphanedDomains)
				catalog.GET("/hosts/orphaned", projectHandler.GetOrphanedHosts)
				catalog.GET("/users/orphaned", projectHandler.GetOrphanedUsers)
				/*catalog.GET("/directory-nodes/orphaned", projectHandler.GetOrphanedDirectoryNodes)

				// --- POST: Manuell zum Katalog hinzufügen (ohne bekannte AD-Position) ---
				catalog.POST("/domains", projectHandler.AddDomainToCatalog)
				catalog.POST("/hosts", projectHandler.AddHostToCatalog)
				catalog.POST("/users", projectHandler.AddUserToCatalog)

				// --- PATCH: Verwaltung / Markierungen ---
				catalog.PATCH("/domains/:uid/high-value", projectHandler.MarkDomainAsHighValue)
				catalog.PATCH("/hosts/:uid/high-value", projectHandler.MarkHostAsHighValue)
				catalog.PATCH("/users/:uid/high-value", projectHandler.MarkUserAsHighValue)

				// Orphaned → Placed (wenn Parent manuell bekannt gemacht wird)
				catalog.PATCH("/domains/:uid/promote", projectHandler.PromoteDomain)
				catalog.PATCH("/hosts/:uid/promote", projectHandler.PromoteHost)
				catalog.PATCH("/users/:uid/promote", projectHandler.PromoteUser)

				// --- DELETE: Aus Katalog entfernen (soft delete) ---
				catalog.DELETE("/domains/:uid", projectHandler.RemoveDomainFromCatalog)
				catalog.DELETE("/hosts/:uid", projectHandler.RemoveHostFromCatalog)
				catalog.DELETE("/users/:uid", projectHandler.RemoveUserFromCatalog)*/
			}

			// --- Active Directories ---
			project.GET("/active-directories", projectHandler.GetProjectActiveDirectories)
			project.POST("/active-directories", projectHandler.AddProjectActiveDirectory)
			project.GET("/active-directories/:adUID", adHandler.GetProjectActiveDirectory)
			project.PATCH("/active-directories/:adUID", adHandler.UpdateProjectActiveDirectory)
			project.GET("/active-directories/:adUID/domains", adHandler.GetActiveDirectoryDomains)
			project.POST("/active-directories/:adUID/domains", adHandler.AddActiveDirectoryDomain)

			// --- Domains ---
			//project.POST("/domains", adHandler.AddActiveDirectoryDomain) TODO implement

			//project.GET("/domains/", projectHandler.GetCatalogDomains)
			project.PATCH("/domains/:domainUID", domainHandler.UpdateDomain)
			project.GET("/domains/:domainUID/changes",
				handlers.EntityType("Domain"),
				changeHandler.GetChanges,
			)
			project.GET("/domains/:domainUID/hosts", domainHandler.GetDomainHosts)
			project.POST("/domains/:domainUID/hosts", domainHandler.AddDomainHost)
			project.GET("/domains/:domainUID/directory-nodes", domainHandler.GetDomainDirectoryNodes)
			project.POST("/domains/:domainUID/directory-nodes", domainHandler.AddDomainDirectoryNode)
			project.GET("/domains/:domainUID/directory-nodes/all", domainHandler.GetDeepChildDirectoryNodes)

			project.GET("/domains/:domainUID/gpos", domainHandler.GetDomainGPOs)
			project.POST("/domains/:domainUID/gpos", domainHandler.LinkDomainGPO)
			project.GET("/domains/:domainUID/gpos/all", domainHandler.GetDomainGPOLib)
			//project.POST("/domains/:domainUID/users", domainHandler.)

			// --- Directory Nodes (OU / Container) ---
			//project.GET("/directory-nodes", projectHandler.GetDirectoryNodes)

			//project.GET("/directory-nodes/details")
			//project.POST("/directory-nodes", dirNodeHandler.CreateDirectoryNode) TODO implement
			/*project.GET("/directory-nodes/:dirNodeUID", dirNodeHandler.GetDirectoryNode)*/
			project.PATCH("/directory-nodes/:dirNodeUID", dirNodeHandler.UpdateDirectoryNode)
			project.GET("/directory-nodes/:dirNodeUID/users", dirNodeHandler.GetDirectoryNodeUsers)
			// TODO change
			project.POST("/directory-nodes/:dirNodeUID/users", dirNodeHandler.GetDirectoryNodeUsers)

			project.POST("/directory-nodes/:dirNodeUID/childs", dirNodeHandler.AddChildDirectoryNode)
			project.GET("/directory-nodes/:dirNodeUID/childs", dirNodeHandler.GetChildDirectoryNodes)

			project.GET("/directory-nodes/:dirNodeUID/childs/all", dirNodeHandler.GetDeepChildDirectoryNodes)

			//TODO IMPLEMENT
			//	project.GET("/directory-nodes/:dirNodeUID/acls", dirNodeHandler.GetACLs)

			// --- Hosts ---
			project.GET("/hosts", projectHandler.GetCatalogHosts)
			project.POST("/hosts", hostHandler.CreateHost)
			project.PATCH("/hosts/:hostUID", hostHandler.UpdateHost)
			project.GET("/hosts/:hostUID/services", serviceHandler.GetServices)
			project.POST("/hosts/:hostUID/services", hostHandler.AddService)
			project.GET("/hosts/:hostUID/changes",
				handlers.EntityType("Host"),
				changeHandler.GetChanges,
			)
			project.POST("/hosts/:hostUID/capabilities", hostHandler.AddCapability)
			project.GET("/hosts/:hostUID/capabilities", hostHandler.GetLinkedCapabilities)

			project.PATCH("/services/:serviceUID", serviceHandler.UpdateService)

			// --- Services ---
			project.GET("/services", projectHandler.GetCatalogServices)

			// --- Users ---
			project.GET("/users", projectHandler.GetCatalogUsers)
			project.POST("/users", userHandler.CreateUser)
			project.PATCH("/users/:userUID", userHandler.UpdateUser)
			project.GET("/users/:userUID/changes",
				handlers.EntityType("User"),
				changeHandler.GetChanges,
			)

			// --- Targets ---
			project.GET("/targets", projectHandler.GetTargets)
			project.POST("/targets", middleware.AddPrefixMiddleware("project"), projectHandler.CreateTarget)

			// --- Logs ---
			project.GET("/logs", logHandler.GetLogs)
			project.POST("/logs/query", logHandler.GetLogsWithOptions)
			project.GET("/logs/types", logHandler.GetLogTypes)
			project.GET("/logs/mkeys", logHandler.GetModuleKeySet)

			// --- Capabilities ---

			project.GET("/capabilities", capabilityHandler.GetCatalogCapabilities)
			//project.GET("")

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
