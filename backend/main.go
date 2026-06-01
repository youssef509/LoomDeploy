package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"loomdeploy/internal/database"
	"loomdeploy/internal/docker"
	"loomdeploy/internal/handlers"
	"loomdeploy/internal/healthcheck"
	"loomdeploy/internal/middleware"
	"loomdeploy/internal/models"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/var/lib/loomdeploy/data.db"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	database.Init(dbPath)
	docker.Init()
	healthcheck.Start()

	// Mark any deployments that were in-progress when the server last stopped as failed.
	now := time.Now()
	var staleMsg = "Deployment interrupted: server restarted before this build completed."
	database.DB.Model(&models.Deployment{}).
		Where("status IN ?", []string{string(models.StatusPending), string(models.StatusBuilding)}).
		Updates(map[string]interface{}{
			"status":      models.StatusFailed,
			"build_logs":  staleMsg,
			"finished_at": now,
		})
	log.Println("Startup: stale deployments cleaned up")

	r := gin.Default()

	allowedOrigins := []string{"http://localhost:3000", "http://localhost:3001"}
	if appDomain := os.Getenv("APP_DOMAIN"); appDomain != "" {
		allowedOrigins = append(allowedOrigins, "https://"+appDomain)
		allowedOrigins = append(allowedOrigins, "http://"+appDomain)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")

	// Public setup-status — tells frontend if first-run setup is needed
	api.GET("/auth/setup-status", handlers.SetupStatus)

	auth := api.Group("/auth")
	{
		// Register is public for first-run; the handler itself enforces admin-only after that
		auth.POST("/register", middleware.OptionalAuth(), handlers.Register)
		auth.POST("/login", handlers.Login)
		auth.GET("/me", middleware.Auth(), handlers.Me)
		auth.POST("/change-password", middleware.Auth(), handlers.ChangePassword)
		auth.GET("/users", middleware.Auth(), handlers.ListUsers)
	}

	system := api.Group("/system", middleware.Auth())
	{
		system.GET("/stats", handlers.GetSystemStats)
	}

	projects := api.Group("/projects", middleware.Auth())
	{
		projects.GET("", handlers.ListProjects)
		projects.POST("", handlers.CreateProject)
		projects.GET("/:id", handlers.GetProject)
		projects.PATCH("/:id", handlers.UpdateProject)
		projects.DELETE("/:id", handlers.DeleteProject)

		projects.POST("/:id/deploy", handlers.DeployProject)
		projects.POST("/:id/container/:action", handlers.ContainerAction)

		projects.GET("/:id/deployments", handlers.ListDeployments)
		projects.POST("/:id/deployments/:depId/redeploy", handlers.RedeployDeployment)
		projects.DELETE("/:id/deployments/:depId", handlers.DeleteDeployment)
		projects.GET("/:id/stats", handlers.ProjectContainerStats)

		api.GET("/deployments/recent", middleware.Auth(), handlers.ListRecentDeployments)

		projects.GET("/:id/env", handlers.ListEnvVars)
		projects.PUT("/:id/env", handlers.SetEnvVars)

		projects.GET("/:id/domains", handlers.ListDomains)
		projects.POST("/:id/domains", handlers.AddDomain)
		projects.DELETE("/:id/domains/:domainId", handlers.RemoveDomain)
		projects.PUT("/:id/domains/primary", handlers.UpdatePrimaryDomain)

		projects.GET("/:id/logs", handlers.StreamContainerLogs)
		projects.GET("/:id/build-logs/stream", handlers.StreamBuildLogs)
	}

	// Public — no auth (secret in URL is the auth token)
	r.POST("/api/webhooks/:secret", handlers.HandleWebhook)

	// Service templates (one-click deploys)
	api.GET("/services/templates", handlers.GetServiceTemplates)
	api.POST("/services", middleware.Auth(), handlers.CreateService)

	// GitHub App push events (HMAC-verified, no auth header needed)
	r.POST("/api/webhooks/github-app", handlers.HandleGitHubAppWebhook)

	// GitHub App public callbacks (under /api/ so Traefik routes them to the backend)
	r.GET("/api/github/app/manifest-callback", handlers.ManifestCallback)
	r.GET("/api/github/app/install-callback", handlers.GitHubInstallCallback)

	// Authenticated project webhook management
	projects.POST("/:id/webhook/regenerate", handlers.RegenerateWebhook)

	settings := api.Group("/settings", middleware.Auth())
	{
		settings.GET("/source-control", handlers.ListSourceTokens)
		settings.PUT("/source-control/:provider", handlers.SetSourceToken)
		settings.DELETE("/source-control/:provider", handlers.DeleteSourceToken)
	}

	// GitHub App — authenticated
	gh := api.Group("/github/app", middleware.Auth())
	{
		gh.GET("/status", handlers.GitHubAppStatus)
		gh.GET("/manifest", handlers.GetManifest)
		gh.GET("/connect", handlers.GitHubAppConnect)
		gh.DELETE("/disconnect", handlers.GitHubAppDisconnect)
		gh.DELETE("/reset", handlers.GitHubAppReset)
		gh.GET("/repos", handlers.ListGitHubRepos)
	}

	log.Printf("LoomDeploy backend starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
