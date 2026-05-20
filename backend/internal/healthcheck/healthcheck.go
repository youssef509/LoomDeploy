package healthcheck

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"loomdeploy/internal/database"
	"loomdeploy/internal/docker"
	"loomdeploy/internal/models"
)

const (
	checkInterval = 30 * time.Second
	maxFailures   = 3
)

var failureCounts sync.Map

func Start() {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()
		for range ticker.C {
			checkAll()
		}
	}()
	log.Println("Health check monitor started (interval: 30s, threshold: 3 failures)")
}

func checkAll() {
	var projects []models.Project
	database.DB.Where("health_check_url != ''").Find(&projects)
	for _, p := range projects {
		go checkProject(p)
	}
}

func checkProject(p models.Project) {
	var dep models.Deployment
	if err := database.DB.Where("project_id = ? AND status = ?", p.ID, models.StatusRunning).
		Order("started_at desc").First(&dep).Error; err != nil || dep.ContainerID == "" {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(p.HealthCheckURL)
	healthy := err == nil && resp != nil && resp.StatusCode < 500
	if resp != nil {
		resp.Body.Close()
	}

	if healthy {
		failureCounts.Delete(p.ID)
		return
	}

	val, _ := failureCounts.LoadOrStore(p.ID, 0)
	count := val.(int) + 1
	failureCounts.Store(p.ID, count)

	log.Printf("[healthcheck] %s FAIL (%s) — %d/%d consecutive failures", p.Name, p.HealthCheckURL, count, maxFailures)

	if count >= maxFailures {
		log.Printf("[healthcheck] Auto-restarting container for project %s after %d failures", p.Name, count)
		failureCounts.Delete(p.ID)
		if err := docker.ContainerAction(context.Background(), dep.ContainerID, "restart"); err != nil {
			log.Printf("[healthcheck] restart failed for %s: %v", p.Name, err)
		}
	}
}
