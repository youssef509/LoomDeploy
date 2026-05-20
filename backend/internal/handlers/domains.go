package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loomdeploy/internal/database"
	"loomdeploy/internal/models"
)

// ListDomains returns all extra domains for a project plus the primary domain.
func ListDomains(c *gin.Context) {
	userID := c.GetString("user_id")
	var project models.Project
	if err := database.DB.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "project not found"})
		return
	}

	var extras []models.ProjectDomain
	database.DB.Where("project_id = ?", project.ID).Order("created_at asc").Find(&extras)

	c.JSON(http.StatusOK, gin.H{
		"primary_domain":     project.Domain,
		"is_generated":       project.IsGeneratedDomain,
		"extra_domains":      extras,
	})
}

// AddDomain attaches an extra custom domain to the project.
func AddDomain(c *gin.Context) {
	userID := c.GetString("user_id")
	var project models.Project
	if err := database.DB.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "project not found"})
		return
	}

	var body struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	pd := models.ProjectDomain{
		ID:        uuid.NewString(),
		ProjectID: project.ID,
		Domain:    body.Domain,
		CreatedAt: time.Now(),
	}
	if err := database.DB.Create(&pd).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"message": "domain already in use"})
		return
	}
	c.JSON(http.StatusCreated, pd)
}

// RemoveDomain detaches an extra domain from the project.
func RemoveDomain(c *gin.Context) {
	userID := c.GetString("user_id")
	var project models.Project
	if err := database.DB.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "project not found"})
		return
	}

	if err := database.DB.
		Where("id = ? AND project_id = ?", c.Param("domainId"), project.ID).
		Delete(&models.ProjectDomain{}).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "domain not found"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// UpdatePrimaryDomain changes or regenerates the primary domain of a project.
func UpdatePrimaryDomain(c *gin.Context) {
	userID := c.GetString("user_id")
	var project models.Project
	if err := database.DB.Where("id = ? AND user_id = ?", c.Param("id"), userID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "project not found"})
		return
	}

	var body struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	updates := map[string]interface{}{
		"domain":              body.Domain,
		"is_generated_domain": false,
	}
	if err := database.DB.Model(&project).Updates(updates).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"message": "domain already in use"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"domain": body.Domain})
}
