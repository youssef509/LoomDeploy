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

func HandleWebhook(c *gin.Context) {
	secret := c.Param("secret")
	if secret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "missing secret"})
		return
	}

	var project models.Project
	if err := database.DB.Where("webhook_secret = ?", secret).First(&project).Error; err != nil {
		// Return 200 to avoid leaking info about valid/invalid secrets
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "failed to read body"})
		return
	}

	// Verify GitHub HMAC-SHA256 signature if present
	if sig := c.GetHeader("X-Hub-Signature-256"); sig != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(sig), []byte(expected)) {
			c.JSON(http.StatusForbidden, gin.H{"message": "invalid signature"})
			return
		}
	}

	// Verify GitLab / Gitea token header if present
	if token := c.GetHeader("X-Gitlab-Token"); token != "" {
		if token != secret {
			c.JSON(http.StatusForbidden, gin.H{"message": "invalid token"})
			return
		}
	}
	if token := c.GetHeader("X-Gitea-Signature"); token != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(token), []byte(expected)) {
			c.JSON(http.StatusForbidden, gin.H{"message": "invalid signature"})
			return
		}
	}

	// Parse ref from payload (GitHub, GitLab, Gitea all use "ref")
	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid or missing ref in payload"})
		return
	}

	// Extract branch: "refs/heads/main" → "main"
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	if branch != project.Branch {
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("push to '%s', project tracks '%s' — skipping", branch, project.Branch),
		})
		return
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

	c.JSON(http.StatusOK, gin.H{"message": "deployment triggered", "deployment_id": dep.ID})
}
