package database

import (
	"log"
	"os"
	"path/filepath"

	"loomdeploy/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(dbPath string) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get raw db: %v", err)
	}
	sqlDB.Exec("PRAGMA foreign_keys = ON;")

	if err := db.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.ProjectDomain{},
		&models.Deployment{},
		&models.EnvVar{},
		&models.SourceToken{},
		&models.GitHubAppCreds{},
	); err != nil {
		log.Fatalf("AutoMigrate failed: %v", err)
	}

	DB = db
	log.Println("Database initialized at", dbPath)

	// Retire stale 'running' records — keep only the newest per project.
	db.Exec(`
		UPDATE deployments
		SET status = 'stopped', finished_at = datetime('now')
		WHERE status = 'running'
		AND id NOT IN (
			SELECT id FROM deployments
			WHERE status = 'running'
			GROUP BY project_id
			HAVING id = MAX(id)
		)
	`)
}
