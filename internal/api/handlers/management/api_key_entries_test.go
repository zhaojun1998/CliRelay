package management

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/usage"
)

func setupAPIKeyEntriesTestDB(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "usage.db")
	if err := usage.InitDB(dbPath, config.RequestLogStorageConfig{}, time.UTC); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(usage.CloseDB)
}

func TestPatchAPIKeyEntryRejectsBlankKeyWithoutDeletingExistingEntry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAPIKeyEntriesTestDB(t)

	if err := usage.UpsertAPIKey(usage.APIKeyRow{
		Key:  "sk-existing-issue-192",
		Name: "Existing Key",
	}); err != nil {
		t.Fatalf("UpsertAPIKey: %v", err)
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPatch,
		"/api-key-entries",
		bytes.NewReader([]byte(`{"index":0,"value":{"key":"   ","name":"Existing Key"}}`)),
	)

	h := NewHandler(&config.Config{}, "", nil)
	h.PatchAPIKeyEntry(c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if got := usage.GetAPIKey("sk-existing-issue-192"); got == nil {
		t.Fatalf("existing key was deleted after blank-key patch")
	}
}

func TestPatchAPIKeyEntryRejectsChangingToExistingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAPIKeyEntriesTestDB(t)

	for _, row := range []usage.APIKeyRow{
		{Key: "sk-original-issue-192", Name: "Original Key"},
		{Key: "sk-target-issue-192", Name: "Target Key"},
	} {
		if err := usage.UpsertAPIKey(row); err != nil {
			t.Fatalf("UpsertAPIKey(%q): %v", row.Key, err)
		}
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPatch,
		"/api-key-entries",
		bytes.NewReader([]byte(`{"match":"sk-original-issue-192","value":{"key":"sk-target-issue-192","name":"Renamed"}}`)),
	)

	h := NewHandler(&config.Config{}, "", nil)
	h.PatchAPIKeyEntry(c)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
	if got := usage.GetAPIKey("sk-original-issue-192"); got == nil || got.Name != "Original Key" {
		t.Fatalf("original key changed unexpectedly: %#v", got)
	}
	if got := usage.GetAPIKey("sk-target-issue-192"); got == nil || got.Name != "Target Key" {
		t.Fatalf("target key changed unexpectedly: %#v", got)
	}
}

func TestPutAPIKeyEntriesPrunesUnknownChannelsBeforeSave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAPIKeyEntriesTestDB(t)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodPut,
		"/api-key-entries",
		bytes.NewReader([]byte(`[{
      "key": "sk-prune-stale-channel",
      "name": "Prune Stale Channel",
      "allowed-channels": ["kimi-A", "kimi-B"]
    }]`)),
	)

	h := NewHandler(&config.Config{
		OpenAICompatibility: []config.OpenAICompatibility{
			{Name: "kimi-B", BaseURL: "https://example.invalid"},
		},
	}, "", nil)
	h.PutAPIKeyEntries(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	got := usage.GetAPIKey("sk-prune-stale-channel")
	if got == nil {
		t.Fatal("expected API key after PUT")
	}
	if containsString(got.AllowedChannels, "kimi-A") {
		t.Fatalf("allowed-channels = %v, should not keep unknown channel", got.AllowedChannels)
	}
	if !containsString(got.AllowedChannels, "kimi-B") {
		t.Fatalf("allowed-channels = %v, should keep known channel", got.AllowedChannels)
	}
}

func TestPatchAPIKeyEntryPrunesUnknownChannelsBeforeSave(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupAPIKeyEntriesTestDB(t)

	if err := usage.UpsertAPIKey(usage.APIKeyRow{
		Key:             "sk-patch-prune",
		Name:            "Patch Prune",
		AllowedChannels: []string{"kimi-A", "kimi-B"},
	}); err != nil {
		t.Fatalf("UpsertAPIKey: %v", err)
	}

	body := []byte(`{
  "match": "sk-patch-prune",
  "value": {
    "allowed-channels": ["kimi-A", "kimi-B"]
  }
}`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPatch, "/api-key-entries", bytes.NewReader(body))

	h := NewHandler(&config.Config{
		OpenAICompatibility: []config.OpenAICompatibility{
			{Name: "kimi-B", BaseURL: "https://example.invalid"},
		},
	}, "", nil)
	h.PatchAPIKeyEntry(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	got := usage.GetAPIKey("sk-patch-prune")
	if got == nil {
		t.Fatal("expected API key after PATCH")
	}
	if containsString(got.AllowedChannels, "kimi-A") {
		t.Fatalf("allowed-channels = %v, should not keep unknown channel", got.AllowedChannels)
	}
	if !containsString(got.AllowedChannels, "kimi-B") {
		t.Fatalf("allowed-channels = %v, should keep known channel", got.AllowedChannels)
	}
}
