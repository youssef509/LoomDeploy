package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loomdeploy/internal/database"
	"loomdeploy/internal/models"
)

type EnvVarField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Default     string `json:"default,omitempty"`
}

type ServiceTemplate struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Icon         string        `json:"icon"`
	Category     string        `json:"category"`
	DockerImage  string        `json:"docker_image"`
	Versions     []string      `json:"versions"`
	DefaultPort  int           `json:"default_port"`
	VolumeMount  string        `json:"volume_mount,omitempty"`
	EnvVarFields []EnvVarField `json:"env_var_fields"`
}

var serviceTemplates = []ServiceTemplate{
	{
		ID:          "postgres",
		Name:        "PostgreSQL",
		Description: "Powerful open-source relational database",
		Icon:        "i-simple-icons-postgresql",
		Category:    "database",
		DockerImage: "postgres:16",
		Versions:    []string{"postgres:16", "postgres:15", "postgres:14", "postgres:13"},
		DefaultPort: 5432,
		VolumeMount: "/var/lib/postgresql/data",
		EnvVarFields: []EnvVarField{
			{Key: "POSTGRES_PASSWORD", Label: "Root Password", Placeholder: "strong-password", Required: true, Secret: true},
			{Key: "POSTGRES_DB", Label: "Database Name", Placeholder: "app", Required: false, Default: "app"},
			{Key: "POSTGRES_USER", Label: "Username", Placeholder: "postgres", Required: false, Default: "postgres"},
		},
	},
	{
		ID:          "mysql",
		Name:        "MySQL",
		Description: "World's most popular open-source database",
		Icon:        "i-simple-icons-mysql",
		Category:    "database",
		DockerImage: "mysql:8",
		Versions:    []string{"mysql:8", "mysql:8.0", "mysql:5.7"},
		DefaultPort: 3306,
		VolumeMount: "/var/lib/mysql",
		EnvVarFields: []EnvVarField{
			{Key: "MYSQL_ROOT_PASSWORD", Label: "Root Password", Placeholder: "strong-password", Required: true, Secret: true},
			{Key: "MYSQL_DATABASE", Label: "Database Name", Placeholder: "app", Required: false},
			{Key: "MYSQL_USER", Label: "Username", Placeholder: "user", Required: false},
			{Key: "MYSQL_PASSWORD", Label: "User Password", Placeholder: "password", Required: false, Secret: true},
		},
	},
	{
		ID:          "mariadb",
		Name:        "MariaDB",
		Description: "Community-developed MySQL-compatible database",
		Icon:        "i-simple-icons-mariadb",
		Category:    "database",
		DockerImage: "mariadb:11",
		Versions:    []string{"mariadb:11", "mariadb:10.11", "mariadb:10.6"},
		DefaultPort: 3306,
		VolumeMount: "/var/lib/mysql",
		EnvVarFields: []EnvVarField{
			{Key: "MARIADB_ROOT_PASSWORD", Label: "Root Password", Placeholder: "strong-password", Required: true, Secret: true},
			{Key: "MARIADB_DATABASE", Label: "Database Name", Placeholder: "app", Required: false},
			{Key: "MARIADB_USER", Label: "Username", Placeholder: "user", Required: false},
			{Key: "MARIADB_PASSWORD", Label: "User Password", Placeholder: "password", Required: false, Secret: true},
		},
	},
	{
		ID:          "redis",
		Name:        "Redis",
		Description: "In-memory data structure store and cache",
		Icon:        "i-simple-icons-redis",
		Category:    "cache",
		DockerImage: "redis:7-alpine",
		Versions:    []string{"redis:7-alpine", "redis:7", "redis:6-alpine"},
		DefaultPort: 6379,
		VolumeMount: "/data",
		EnvVarFields: []EnvVarField{},
	},
	{
		ID:          "mongodb",
		Name:        "MongoDB",
		Description: "Document-oriented NoSQL database",
		Icon:        "i-simple-icons-mongodb",
		Category:    "database",
		DockerImage: "mongo:7",
		Versions:    []string{"mongo:7", "mongo:6", "mongo:5"},
		DefaultPort: 27017,
		VolumeMount: "/data/db",
		EnvVarFields: []EnvVarField{
			{Key: "MONGO_INITDB_ROOT_USERNAME", Label: "Root Username", Placeholder: "admin", Required: false, Default: "admin"},
			{Key: "MONGO_INITDB_ROOT_PASSWORD", Label: "Root Password", Placeholder: "strong-password", Required: false, Secret: true},
			{Key: "MONGO_INITDB_DATABASE", Label: "Initial Database", Placeholder: "app", Required: false},
		},
	},
	{
		ID:          "wordpress",
		Name:        "WordPress",
		Description: "Popular CMS powering millions of websites",
		Icon:        "i-simple-icons-wordpress",
		Category:    "cms",
		DockerImage: "wordpress:latest",
		Versions:    []string{"wordpress:latest", "wordpress:6", "wordpress:php8.3"},
		DefaultPort: 80,
		VolumeMount: "/var/www/html",
		EnvVarFields: []EnvVarField{
			{Key: "WORDPRESS_DB_HOST", Label: "DB Host", Placeholder: "mysql-container:3306", Required: true},
			{Key: "WORDPRESS_DB_USER", Label: "DB User", Placeholder: "wordpress", Required: true},
			{Key: "WORDPRESS_DB_PASSWORD", Label: "DB Password", Placeholder: "password", Required: true, Secret: true},
			{Key: "WORDPRESS_DB_NAME", Label: "DB Name", Placeholder: "wordpress", Required: true, Default: "wordpress"},
		},
	},
	{
		ID:          "ghost",
		Name:        "Ghost",
		Description: "Professional publishing platform",
		Icon:        "i-simple-icons-ghost",
		Category:    "cms",
		DockerImage: "ghost:5-alpine",
		Versions:    []string{"ghost:5-alpine", "ghost:5", "ghost:4-alpine"},
		DefaultPort: 2368,
		VolumeMount: "/var/lib/ghost/content",
		EnvVarFields: []EnvVarField{
			{Key: "url", Label: "Site URL", Placeholder: "https://myblog.example.com", Required: true},
			{Key: "NODE_ENV", Label: "Environment", Placeholder: "production", Required: false, Default: "production"},
		},
	},
	{
		ID:          "minio",
		Name:        "MinIO",
		Description: "S3-compatible object storage",
		Icon:        "i-simple-icons-minio",
		Category:    "storage",
		DockerImage: "minio/minio:latest",
		Versions:    []string{"minio/minio:latest"},
		DefaultPort: 9000,
		VolumeMount: "/data",
		EnvVarFields: []EnvVarField{
			{Key: "MINIO_ROOT_USER", Label: "Root User", Placeholder: "minioadmin", Required: true, Default: "minioadmin"},
			{Key: "MINIO_ROOT_PASSWORD", Label: "Root Password", Placeholder: "strong-password", Required: true, Secret: true},
		},
	},
}

// GetServiceTemplates GET /api/services/templates
func GetServiceTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, serviceTemplates)
}

type createServiceRequest struct {
	TemplateID    string            `json:"template_id" binding:"required"`
	Name          string            `json:"name" binding:"required"`
	Image         string            `json:"image" binding:"required"`
	Domain        string            `json:"domain"`
	VolumeEnabled bool              `json:"volume_enabled"`
	EnvVars       []envVarKV        `json:"env_vars"`
}

type envVarKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CreateService POST /api/services
func CreateService(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID, _ := userIDVal.(string)

	var req createServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var tmpl *ServiceTemplate
	for i, t := range serviceTemplates {
		if t.ID == req.TemplateID {
			tmpl = &serviceTemplates[i]
			break
		}
	}
	if tmpl == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "unknown template: " + req.TemplateID})
		return
	}

	domain := req.Domain
	if domain == "" {
		domain = autoGenerateDomain(req.Name)
	}

	volumeMount := ""
	if req.VolumeEnabled {
		volumeMount = tmpl.VolumeMount
	}

	projectID := uuid.NewString()
	project := models.Project{
		ID:            projectID,
		UserID:        userID,
		Name:          req.Name,
		SourceType:    models.SourceImage,
		DockerImage:   req.Image,
		Domain:        domain,
		IsGeneratedDomain: req.Domain == "",
		ContainerPort: tmpl.DefaultPort,
		VolumeMount:   volumeMount,
		CreatedAt:     time.Now(),
	}

	if err := database.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create service: " + err.Error()})
		return
	}

	var envVarModels []models.EnvVar
	for _, kv := range req.EnvVars {
		if kv.Key == "" {
			continue
		}
		envVarModels = append(envVarModels, models.EnvVar{
			ID:        uuid.NewString(),
			ProjectID: projectID,
			Key:       kv.Key,
			Value:     kv.Value,
		})
	}
	if len(envVarModels) > 0 {
		database.DB.Create(&envVarModels)
	}

	dep := models.Deployment{
		ID:        uuid.NewString(),
		ProjectID: projectID,
		Status:    models.StatusPending,
		StartedAt: time.Now(),
	}
	database.DB.Create(&dep)

	startDeployGoroutine(dep, project, envVarModels, "")

	c.JSON(http.StatusCreated, gin.H{
		"project":    project,
		"deployment": dep,
	})
}
