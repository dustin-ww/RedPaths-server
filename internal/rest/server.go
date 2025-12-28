package rest

import (
	"RedPaths-server/pkg/module_exec"
	"RedPaths-server/pkg/service"
	"RedPaths-server/pkg/service/active_directory"
	"RedPaths-server/pkg/service/redpaths"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var logger *slog.Logger

func StartServer(port string, postgresCon *gorm.DB, dgraphCon *dgo.Dgraph) {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4000", "http://localhost:4000/"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Use(func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("Content-Security-Policy", "default-src 'self'; connect-src *; font-src *; script-src-elem * 'unsafe-inline'; img-src * data:; style-src * 'unsafe-inline';")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		c.Header("Referrer-Policy", "strict-origin")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Permissions-Policy", "geolocation=(),midi=(),sync-xhr=(),microphone=(),camera=(),magnetometer=(),gyroscope=(),fullscreen=(self),payment=()")
		c.Next()
	})

	if err := initLogger(); err != nil {
		panic(err)
	}

	projectService, err := active_directory.NewProjectService(dgraphCon, postgresCon)
	domainService, err := active_directory.NewDomainService(dgraphCon)
	hostService, err := active_directory.NewHostService(dgraphCon)
	serviceService, err := active_directory.NewServiceService(dgraphCon)
	userService, err := active_directory.NewUserService(dgraphCon)
	moduleExecutor := module_exec.GlobalRegistry
	redPathsModuleService, err := redpaths.NewModuleService(moduleExecutor, moduleExecutor.RecommendationEngine, postgresCon)
	logService, err := service.NewLogService(postgresCon)
	if err != nil {
		log.Fatalf("Failed to initialize ProjectService: %v", err)
	}
	RegisterProjectHandlers(router, projectService, logService, domainService, hostService, serviceService, userService)
	RegisterRedPathsModuleHandlers(router, redPathsModuleService, projectService)
	RegisterServerHandlers(router)
	logger.Info("Starting server")

	fmt.Println("Starting SSE server...")
	router.Run(":" + port)
}

func initLogger() error {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, file)

	logger = slog.New(slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(logger)
	return nil
}
