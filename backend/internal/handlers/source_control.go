package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"loomdeploy/internal/database"
	"loomdeploy/internal/models"
)

type setSourceTokenRequest struct {
	Token string `json:"token" binding:"required"`
	Label string `json:"label"`
}

func ListSourceTokens(c *gin.Context) {
	var tokens []models.SourceToken
	database.DB.Find(&tokens)
	for i := range tokens {
		tokens[i].HasToken = tokens[i].Token != ""
	}
	c.JSON(http.StatusOK, tokens)
}

func SetSourceToken(c *gin.Context) {
	provider := c.Param("provider")
	var req setSourceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if req.Label == "" {
		req.Label = provider
	}

	token := models.SourceToken{
		Provider:  provider,
		Label:     req.Label,
		Token:     req.Token,
		UpdatedAt: time.Now(),
	}
	if err := database.DB.Save(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save token"})
		return
	}
	token.HasToken = true
	c.JSON(http.StatusOK, token)
}

func DeleteSourceToken(c *gin.Context) {
	provider := c.Param("provider")
	database.DB.Delete(&models.SourceToken{}, "provider = ?", provider)
	c.JSON(http.StatusOK, gin.H{"message": "token removed"})
}
