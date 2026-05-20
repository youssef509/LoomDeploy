package models

import "time"

type UserRole string

const (
	RoleAdmin     UserRole = "admin"
	RoleDeveloper UserRole = "developer"
	RoleViewer    UserRole = "viewer"
)

type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         UserRole  `gorm:"default:admin" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

type SourceType string

const (
	SourceGit    SourceType = "git"
	SourceUpload SourceType = "upload"
)

type Project struct {
	ID                     string          `gorm:"primaryKey" json:"id"`
	UserID                 string          `gorm:"not null" json:"user_id"`
	Name                   string          `gorm:"not null" json:"name"`
	SourceType             SourceType      `gorm:"default:git" json:"source_type"`
	RepositoryURL          string          `json:"repository_url,omitempty"`
	Branch                 string          `gorm:"default:main" json:"branch,omitempty"`
	Domain                 string          `gorm:"uniqueIndex;not null" json:"domain"`
	IsGeneratedDomain      bool            `gorm:"column:is_generated_domain;default:false" json:"is_generated_domain"`
	ContainerPort          int             `gorm:"default:3000" json:"container_port"`
	GitToken               string          `gorm:"column:git_token" json:"-"`
	HasGitToken            bool            `gorm:"-" json:"has_git_token"`
	WebhookSecret          string          `gorm:"column:webhook_secret" json:"webhook_secret,omitempty"`
	HasWebhook             bool            `gorm:"-" json:"has_webhook"`
	NotificationWebhookURL string          `gorm:"column:notification_webhook_url" json:"notification_webhook_url,omitempty"`
	CPULimit               float64         `gorm:"default:0" json:"cpu_limit"`
	MemoryLimitMB          int             `gorm:"column:memory_limit_mb;default:0" json:"memory_limit_mb"`
	HealthCheckURL         string          `gorm:"column:health_check_url" json:"health_check_url,omitempty"`
	DockerfileContent      string          `gorm:"column:dockerfile_content;type:text" json:"dockerfile_content,omitempty"`
	CreatedAt              time.Time       `json:"created_at"`
	Deployments            []Deployment    `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"-"`
	EnvVars                []EnvVar        `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"-"`
	ExtraDomains           []ProjectDomain `gorm:"foreignKey:ProjectID;constraint:OnDelete:CASCADE" json:"extra_domains,omitempty"`
	LatestDeployment       *Deployment     `gorm:"-" json:"latest_deployment,omitempty"`
}

// ProjectDomain stores additional custom domains attached to a project.
// The primary domain lives on Project.Domain.
type ProjectDomain struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	ProjectID string    `gorm:"not null;index" json:"project_id"`
	Domain    string    `gorm:"uniqueIndex;not null" json:"domain"`
	CreatedAt time.Time `json:"created_at"`
}

type DeploymentStatus string

const (
	StatusPending  DeploymentStatus = "pending"
	StatusBuilding DeploymentStatus = "building"
	StatusRunning  DeploymentStatus = "running"
	StatusFailed   DeploymentStatus = "failed"
	StatusStopped  DeploymentStatus = "stopped"
)

type Deployment struct {
	ID          string           `gorm:"primaryKey" json:"id"`
	ProjectID   string           `gorm:"not null;index" json:"project_id"`
	Status      DeploymentStatus `gorm:"default:pending" json:"status"`
	ContainerID string           `json:"container_id,omitempty"`
	CommitSHA   string           `json:"commit_sha,omitempty"`
	BuildLogs   string           `gorm:"type:text" json:"build_logs,omitempty"`
	StartedAt   time.Time        `json:"started_at"`
	FinishedAt  *time.Time       `json:"finished_at,omitempty"`
}

type EnvVar struct {
	ID        string `gorm:"primaryKey" json:"id"`
	ProjectID string `gorm:"not null;index" json:"project_id"`
	Key       string `gorm:"not null" json:"key"`
	Value     string `gorm:"not null" json:"value"`
}

// GitHubAppCreds holds the GitHub App credentials created via the manifest flow.
// Uses a singleton record (ID=1). Replaces GITHUB_APP_* env vars once set up.
type GitHubAppCreds struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	AppID         string `gorm:"not null" json:"app_id"`
	AppName       string `gorm:"not null" json:"app_name"`
	ClientID      string `json:"-"`
	ClientSecret  string `gorm:"type:text" json:"-"`
	PrivateKey    string `gorm:"type:text" json:"-"`
	WebhookSecret string `json:"-"`
}

type SourceToken struct {
	Provider       string    `gorm:"primaryKey" json:"provider"`
	Label          string    `json:"label"`
	Token          string    `gorm:"not null" json:"-"`
	HasToken       bool      `gorm:"-" json:"has_token"`
	InstallationID int64     `gorm:"column:installation_id;default:0" json:"installation_id,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}
