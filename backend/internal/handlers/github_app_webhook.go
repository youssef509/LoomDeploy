package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loomdeploy/internal/database"
	"loomdeploy/internal/models"
)

// HandleGitHubAppWebhook POST /api/webhooks/github-app — public
// Receives push events from the GitHub App and triggers matching deployments.
func HandleGitHubAppWebhook(c *gin.Context) {
	event := c.GetHeader("X-GitHub-Event")

	// GitHub sends a ping when the webhook is first configured
	if event == "ping" {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
		return
	}

	if event != "push" {
		c.JSON(http.StatusOK, gin.H{"message": "event ignored"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to read body"})
		return
	}

	// Verify HMAC-SHA256 signature using the stored GitHub App webhook secret
	if sig := c.GetHeader("X-Hub-Signature-256"); sig != "" {
		creds, hasCreds := getDBCreds()
		if hasCreds && creds.WebhookSecret != "" {
			mac := hmac.New(sha256.New, []byte(creds.WebhookSecret))
			mac.Write(body)
			expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
			if !hmac.Equal([]byte(sig), []byte(expected)) {
				c.JSON(http.StatusForbidden, gin.H{"message": "invalid signature"})
				return
			}
		}
	}

	var payload struct {
		Ref        string `json:"ref"`
		Repository struct {
			CloneURL string `json:"clone_url"`
			HTMLURL  string `json:"html_url"`
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid or missing payload"})
		return
	}

	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")

	// Normalise repo URLs for comparison (strip .git, lowercase)
	normalise := func(u string) string {
		return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(u)), ".git")
	}
	cloneURL := normalise(payload.Repository.CloneURL)
	htmlURL := normalise(payload.Repository.HTMLURL)

	// Find all projects tracking this repo + branch
	var projects []models.Project
	database.DB.Where("branch = ?", branch).Find(&projects)

	triggered := 0
	for _, project := range projects {
		projURL := normalise(project.RepositoryURL)
		if projURL != cloneURL && projURL != htmlURL {
			continue
		}

		var envVars []models.EnvVar
		database.DB.Where("project_id = ?", project.ID).Find(&envVars)

		dep := models.Deployment{
			ID:        uuid.NewString(),
			ProjectID: project.ID,
			Status:    models.StatusPending,
			StartedAt: time.Now(),
		}
		database.DB.Create(&dep)
		startDeployGoroutine(dep, project, envVars, "")
		triggered++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("triggered %d deployment(s)", triggered),
		"repo":      payload.Repository.FullName,
		"branch":    branch,
		"triggered": triggered,
	})
}
