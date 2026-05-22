package handlers

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"loomdeploy/internal/database"
	"loomdeploy/internal/models"
)

const ghAppProvider = "github_app"

// ── credential helpers (DB-first, env-var fallback) ─────────────────────────

func getDBCreds() (*models.GitHubAppCreds, bool) {
	var c models.GitHubAppCreds
	if database.DB.First(&c).Error == nil && c.AppID != "" {
		return &c, true
	}
	return nil, false
}

func parsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	raw = strings.ReplaceAll(raw, `\n`, "\n")
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func generateGitHubAppJWT() (string, error) {
	var appID string
	var key *rsa.PrivateKey

	if creds, ok := getDBCreds(); ok {
		appID = creds.AppID
		var err error
		key, err = parsePrivateKey(creds.PrivateKey)
		if err != nil {
			return "", fmt.Errorf("DB private key: %w", err)
		}
	} else {
		appID = os.Getenv("GITHUB_APP_ID")
		raw := os.Getenv("GITHUB_APP_PRIVATE_KEY")
		if appID == "" || raw == "" {
			return "", fmt.Errorf("no GitHub App credentials configured")
		}
		var err error
		key, err = parsePrivateKey(raw)
		if err != nil {
			return "", fmt.Errorf("env private key: %w", err)
		}
	}

	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": appID,
	})
	return tok.SignedString(key)
}

func getInstallationAccessToken(installationID int64) (string, error) {
	appJWT, err := generateGitHubAppJWT()
	if err != nil {
		return "", fmt.Errorf("jwt: %w", err)
	}
	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)
	req, _ := http.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Token == "" {
		return "", fmt.Errorf("empty token from GitHub API (status %d)", resp.StatusCode)
	}
	return result.Token, nil
}

// GetGitHubCloneToken returns a short-lived installation token for git clone.
func GetGitHubCloneToken() (string, error) {
	var st models.SourceToken
	if err := database.DB.Where("provider = ? AND installation_id > 0", ghAppProvider).First(&st).Error; err != nil {
		return "", nil
	}
	return getInstallationAccessToken(st.InstallationID)
}

// ── manifest helpers ─────────────────────────────────────────────────────────

func frontendURL(path, query string) string {
	base := "https://" + os.Getenv("APP_DOMAIN")
	u := base + path
	if query != "" {
		u += "?" + query
	}
	return u
}

// ── HTTP handlers ─────────────────────────────────────────────────────────────

// GitHubAppStatus GET /api/github/app/status
// Returns: { phase: "unconfigured" | "setup" | "connected", app_name?, app_id?, installation_id? }
func GitHubAppStatus(c *gin.Context) {
	creds, hasCreds := getDBCreds()

	// Fallback: env vars (legacy / manual setup)
	if !hasCreds {
		appID := os.Getenv("GITHUB_APP_ID")
		appName := os.Getenv("GITHUB_APP_NAME")
		if appID == "" || appName == "" {
			c.JSON(http.StatusOK, gin.H{"phase": "unconfigured"})
			return
		}
		var st models.SourceToken
		connected := database.DB.Where("provider = ?", ghAppProvider).First(&st).Error == nil && st.InstallationID > 0
		phase := "setup"
		if connected {
			phase = "connected"
		}
		c.JSON(http.StatusOK, gin.H{"phase": phase, "app_name": appName, "app_id": appID, "installation_id": st.InstallationID})
		return
	}

	var st models.SourceToken
	connected := database.DB.Where("provider = ?", ghAppProvider).First(&st).Error == nil && st.InstallationID > 0
	phase := "setup"
	if connected {
		phase = "connected"
	}
	c.JSON(http.StatusOK, gin.H{
		"phase":           phase,
		"app_name":        creds.AppName,
		"app_id":          creds.AppID,
		"installation_id": st.InstallationID,
	})
}

// GetManifest GET /api/github/app/manifest — authenticated
// Returns the manifest JSON the frontend will POST to GitHub.
func GetManifest(c *gin.Context) {
	appDomain := os.Getenv("APP_DOMAIN")
	if appDomain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "APP_DOMAIN env var is not set"})
		return
	}
	base := "https://" + appDomain
	manifest := gin.H{
		"name":        "LoomDeploy",
		"url":         base,
		"description": "LoomDeploy self-hosted PaaS",
		"hook_attributes": gin.H{
			"url":    base + "/api/webhooks/github-app",
			"active": true,
		},
		"redirect_url":     base + "/api/github/app/manifest-callback",
		"setup_url":        base + "/api/github/app/install-callback",
		"setup_on_update":  true,
		"callback_urls":    []string{base + "/api/github/app/install-callback"},
		"public":           false,
		"default_permissions": gin.H{
			"contents": "read",
			"metadata": "read",
		},
		"default_events": []string{"push"},
	}
	c.JSON(http.StatusOK, manifest)
}

// ManifestCallback GET /api/github/app/manifest-callback — public
// GitHub redirects here after creating the app. Exchanges code for credentials.
func ManifestCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusFound, frontendURL("/settings/source-control", "gh_error=missing_code"))
		return
	}

	url := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, _ := http.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		log.Printf("manifest conversion: status=%d err=%v", resp.StatusCode, err)
		c.Redirect(http.StatusFound, frontendURL("/settings/source-control", "gh_error=exchange_failed"))
		return
	}
	defer resp.Body.Close()

	var result struct {
		ID            int64  `json:"id"`
		Slug          string `json:"slug"`
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
		PEM           string `json:"pem"`
		WebhookSecret string `json:"webhook_secret"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.ID == 0 {
		c.Redirect(http.StatusFound, frontendURL("/settings/source-control", "gh_error=decode_failed"))
		return
	}

	creds := models.GitHubAppCreds{
		ID:            1,
		AppID:         fmt.Sprintf("%d", result.ID),
		AppName:       result.Slug,
		ClientID:      result.ClientID,
		ClientSecret:  result.ClientSecret,
		PrivateKey:    result.PEM,
		WebhookSecret: result.WebhookSecret,
	}
	if err := database.DB.Save(&creds).Error; err != nil {
		log.Printf("failed to save GitHubAppCreds: %v", err)
		c.Redirect(http.StatusFound, frontendURL("/settings/source-control", "gh_error=db_save"))
		return
	}
	log.Printf("GitHub App registered via manifest: id=%d slug=%s", result.ID, result.Slug)

	// Send user to install the app on their account
	c.Redirect(http.StatusFound, "https://github.com/apps/"+result.Slug+"/installations/new")
}

// GitHubInstallCallback GET /api/github/app/install-callback — public
// GitHub redirects here after installation (setup_url in manifest).
func GitHubInstallCallback(c *gin.Context) {
	idStr := c.Query("installation_id")
	if idStr == "" {
		// No installation_id means user cancelled — go back to settings
		c.Redirect(http.StatusFound, frontendURL("/settings/source-control", ""))
		return
	}
	installationID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.Redirect(http.StatusFound, frontendURL("/settings/source-control", "gh_error=invalid_installation_id"))
		return
	}

	st := models.SourceToken{
		Provider:       ghAppProvider,
		Label:          "GitHub App",
		Token:          "-",
		InstallationID: installationID,
		UpdatedAt:      time.Now(),
	}
	if err := database.DB.Save(&st).Error; err != nil {
		log.Printf("install callback save error: %v", err)
		c.Redirect(http.StatusFound, frontendURL("/settings/source-control", "gh_error=db_save"))
		return
	}
	log.Printf("GitHub App installed: installation_id=%d", installationID)
	c.Redirect(http.StatusFound, frontendURL("/settings/source-control", "gh_connected=1"))
}

// GitHubAppDisconnect DELETE /api/github/app/disconnect — removes installation only
func GitHubAppDisconnect(c *gin.Context) {
	database.DB.Delete(&models.SourceToken{}, "provider = ?", ghAppProvider)
	c.JSON(http.StatusOK, gin.H{"message": "installation removed"})
}

// GitHubAppReset DELETE /api/github/app/reset — removes app credentials + installation
func GitHubAppReset(c *gin.Context) {
	database.DB.Delete(&models.SourceToken{}, "provider = ?", ghAppProvider)
	database.DB.Delete(&models.GitHubAppCreds{}, "id = 1")
	c.JSON(http.StatusOK, gin.H{"message": "GitHub App configuration removed"})
}

// GitHubAppConnect GET /api/github/app/connect — redirect to install page (legacy / env-var flow)
func GitHubAppConnect(c *gin.Context) {
	appName := ""
	if creds, ok := getDBCreds(); ok {
		appName = creds.AppName
	} else {
		appName = os.Getenv("GITHUB_APP_NAME")
	}
	if appName == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "GitHub App not configured"})
		return
	}
	c.Redirect(http.StatusFound, "https://github.com/apps/"+appName+"/installations/new")
}

// ListGitHubRepos GET /api/github/app/repos
func ListGitHubRepos(c *gin.Context) {
	var st models.SourceToken
	if err := database.DB.Where("provider = ?", ghAppProvider).First(&st).Error; err != nil || st.InstallationID == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "GitHub App not connected"})
		return
	}
	token, err := getInstallationAccessToken(st.InstallationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	type Repo struct {
		FullName      string `json:"full_name"`
		CloneURL      string `json:"clone_url"`
		DefaultBranch string `json:"default_branch"`
		Private       bool   `json:"private"`
	}
	var all []Repo
	client := &http.Client{Timeout: 15 * time.Second}
	for page := 1; ; page++ {
		url := fmt.Sprintf("https://api.github.com/installation/repositories?per_page=100&page=%d", page)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		resp, err := client.Do(req)
		if err != nil {
			break
		}
		var result struct {
			Repositories []Repo `json:"repositories"`
			TotalCount   int    `json:"total_count"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		all = append(all, result.Repositories...)
		if len(all) >= result.TotalCount || len(result.Repositories) == 0 {
			break
		}
	}
	if all == nil {
		all = []Repo{}
	}
	c.JSON(http.StatusOK, all)
}
