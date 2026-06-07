package management

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/access"
	configaccess "github.com/router-for-me/CLIProxyAPI/v6/internal/access/config_access"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	apikeysettings "github.com/router-for-me/CLIProxyAPI/v6/internal/management/settings/apikey"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

// refreshAPIKeyCache rebuilds the in-memory access provider cache from SQLite.
// Must be called after every API key write operation.
func (h *Handler) refreshAPIKeyCache() {
	if h == nil || h.cfg == nil {
		return
	}
	// Always update the global provider registry (used during config reload and service bootstrap).
	configaccess.Register(&h.cfg.SDKConfig)
	// Also update the live access manager provider snapshot so changes take effect immediately
	// without waiting for a full config reload.
	if h.accessManager != nil {
		_, _ = access.ApplyAccessProviders(h.accessManager, nil, h.cfg)
	}
}

func (h *Handler) apiKeySettings() *apikeysettings.Service {
	if h == nil {
		return apikeysettings.NewService(nil, nil, nil)
	}
	validateEntry := func(entry config.APIKeyEntry) error {
		var auths []*coreauth.Auth
		if h.authManager != nil {
			auths = h.authManager.List()
		}
		routingCfg := config.RoutingConfig{}
		if h.cfg != nil {
			routingCfg = h.cfg.Routing
		}
		return validateRoutingAndAPIKeyRestrictions(&config.Config{
			SDKConfig: config.SDKConfig{
				APIKeyEntries: []config.APIKeyEntry{entry},
			},
			Routing: routingCfg,
		}, auths)
	}
	return apikeysettings.NewService(h.sanitizeAllowedChannelsForSave, h.validateAllowedChannelGroups, validateEntry)
}

// api-keys (legacy simple list — now backed by SQLite)
func (h *Handler) GetAPIKeys(c *gin.Context) {
	c.JSON(200, gin.H{"api-keys": h.apiKeySettings().EnabledKeys()})
}

func (h *Handler) PutAPIKeys(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read body"})
		return
	}
	var arr []string
	if err = json.Unmarshal(data, &arr); err != nil {
		var obj struct {
			Items []string `json:"items"`
		}
		if err2 := json.Unmarshal(data, &obj); err2 != nil || len(obj.Items) == 0 {
			c.JSON(400, gin.H{"error": "invalid body"})
			return
		}
		arr = obj.Items
	}
	if err := h.apiKeySettings().ReplaceKeys(arr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.refreshAPIKeyCache()
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) PatchAPIKeys(c *gin.Context) {
	var body struct {
		Old *string `json:"old"`
		New *string `json:"new"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Old == nil || body.New == nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}
	if err := h.apiKeySettings().PatchKey(*body.Old, *body.New); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.refreshAPIKeyCache()
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) DeleteAPIKeys(c *gin.Context) {
	if err := h.apiKeySettings().DeleteKey(c.Query("value")); err != nil {
		if errors.Is(err, apikeysettings.ErrMissingValue) {
			c.JSON(400, gin.H{"error": "missing value"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.refreshAPIKeyCache()
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) GetAPIKeyPermissionProfiles(c *gin.Context) {
	profiles := h.apiKeySettings().PermissionProfiles()
	c.JSON(200, gin.H{
		"api-key-permission-profiles": profiles,
		"items":                       profiles,
	})
}

func (h *Handler) PutAPIKeyPermissionProfiles(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read body"})
		return
	}

	var profiles []usage.APIKeyPermissionProfileRow
	if err = json.Unmarshal(data, &profiles); err != nil {
		var obj struct {
			Items []usage.APIKeyPermissionProfileRow `json:"items"`
		}
		if err2 := json.Unmarshal(data, &obj); err2 != nil {
			c.JSON(400, gin.H{"error": "invalid body"})
			return
		}
		profiles = obj.Items
	}

	if err := h.apiKeySettings().ReplacePermissionProfiles(profiles); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.refreshAPIKeyCache()
	c.JSON(200, gin.H{"status": "ok"})
}

// api-key-entries: backed by SQLite api_keys table
func (h *Handler) GetAPIKeyEntries(c *gin.Context) {
	c.JSON(200, gin.H{"api-key-entries": h.apiKeySettings().APIKeyEntries()})
}

func (h *Handler) PutAPIKeyEntries(c *gin.Context) {
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read body"})
		return
	}
	var arr []config.APIKeyEntry
	if err = json.Unmarshal(data, &arr); err != nil {
		var obj struct {
			Items []config.APIKeyEntry `json:"items"`
		}
		if err2 := json.Unmarshal(data, &obj); err2 != nil || len(obj.Items) == 0 {
			c.JSON(400, gin.H{"error": "invalid body"})
			return
		}
		arr = obj.Items
	}
	if err := h.apiKeySettings().ReplaceEntries(arr); err != nil {
		if errors.Is(err, apikeysettings.ErrInvalidEntry) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.refreshAPIKeyCache()
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) PatchAPIKeyEntry(c *gin.Context) {
	var body struct {
		Index *int                             `json:"index"`
		Match *string                          `json:"match"`
		Value *apikeysettings.APIKeyEntryPatch `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Value == nil {
		c.JSON(400, gin.H{"error": "invalid body"})
		return
	}
	if err := h.apiKeySettings().PatchEntry(body.Index, body.Match, *body.Value); err != nil {
		switch {
		case errors.Is(err, apikeysettings.ErrItemNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		case errors.Is(err, apikeysettings.ErrKeyRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "key is required"})
			return
		case errors.Is(err, apikeysettings.ErrAPIKeyExists):
			c.JSON(http.StatusConflict, gin.H{"error": "api key already exists"})
			return
		case errors.Is(err, apikeysettings.ErrInvalidEntry):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.refreshAPIKeyCache()
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) DeleteAPIKeyEntry(c *gin.Context) {
	deleteLogs := shouldDeleteAPIKeyLogs(c)
	var index *int
	if idxStr := c.Query("index"); idxStr != "" {
		parsed, err := strconv.Atoi(idxStr)
		if err == nil {
			index = &parsed
		}
	}
	deletedKey, err := h.apiKeySettings().DeleteEntry(c.Query("key"), index)
	if err != nil {
		if errors.Is(err, apikeysettings.ErrMissingValue) {
			c.JSON(400, gin.H{"error": "missing key or index"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var logsDeleted int64
	if deleteLogs && deletedKey != "" {
		logsDeleted, _ = usage.DeleteLogsByAPIKey(deletedKey)
	}
	h.refreshAPIKeyCache()
	c.JSON(200, gin.H{"status": "ok", "logs_deleted": logsDeleted})
}

func shouldDeleteAPIKeyLogs(c *gin.Context) bool {
	raw := strings.TrimSpace(c.Query("delete_logs"))
	if raw == "" {
		return true
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return true
	}
	return value
}
