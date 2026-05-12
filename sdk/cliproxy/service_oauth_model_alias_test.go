package cliproxy

import (
	"testing"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
)

func TestApplyOAuthModelAlias_Rename(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5"},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 1 {
		t.Fatalf("expected 1 model, got %d", len(out))
	}
	if out[0].ID != "g5" {
		t.Fatalf("expected model id %q, got %q", "g5", out[0].ID)
	}
	if out[0].Name != "models/g5" {
		t.Fatalf("expected model name %q, got %q", "models/g5", out[0].Name)
	}
	if out[0].OriginalID != "gpt-5" {
		t.Fatalf("expected original id %q, got %q", "gpt-5", out[0].OriginalID)
	}
}

func TestApplyOAuthModelAlias_ForkAddsAlias(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5", Fork: true},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 2 {
		t.Fatalf("expected 2 models, got %d", len(out))
	}
	if out[0].ID != "gpt-5" {
		t.Fatalf("expected first model id %q, got %q", "gpt-5", out[0].ID)
	}
	if out[1].ID != "g5" {
		t.Fatalf("expected second model id %q, got %q", "g5", out[1].ID)
	}
	if out[1].Name != "models/g5" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5", out[1].Name)
	}
	if out[1].OriginalID != "gpt-5" {
		t.Fatalf("expected forked original id %q, got %q", "gpt-5", out[1].OriginalID)
	}
}

func TestApplyOAuthModelAlias_ForkAddsMultipleAliases(t *testing.T) {
	cfg := &config.Config{
		OAuthModelAlias: map[string][]config.OAuthModelAlias{
			"codex": {
				{Name: "gpt-5", Alias: "g5", Fork: true},
				{Name: "gpt-5", Alias: "g5-2", Fork: true},
			},
		},
	}
	models := []*ModelInfo{
		{ID: "gpt-5", Name: "models/gpt-5"},
	}

	out := applyOAuthModelAlias(cfg, "codex", "oauth", models)
	if len(out) != 3 {
		t.Fatalf("expected 3 models, got %d", len(out))
	}
	if out[0].ID != "gpt-5" {
		t.Fatalf("expected first model id %q, got %q", "gpt-5", out[0].ID)
	}
	if out[1].ID != "g5" {
		t.Fatalf("expected second model id %q, got %q", "g5", out[1].ID)
	}
	if out[1].Name != "models/g5" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5", out[1].Name)
	}
	if out[2].ID != "g5-2" {
		t.Fatalf("expected third model id %q, got %q", "g5-2", out[2].ID)
	}
	if out[2].Name != "models/g5-2" {
		t.Fatalf("expected forked model name %q, got %q", "models/g5-2", out[2].Name)
	}
	if out[2].OriginalID != "gpt-5" {
		t.Fatalf("expected forked original id %q, got %q", "gpt-5", out[2].OriginalID)
	}
}

func TestBuildConfigModelsRecordsOriginalID(t *testing.T) {
	models := buildConfigModels([]internalconfig.ClaudeModel{
		{Name: "mimo-v2.5-pro", Alias: "claude-opus-4-6"},
		{Name: "claude-opus-4-6", Alias: "claude-opus-4-6"},
	}, "anthropic", "claude")

	if len(models) != 1 {
		t.Fatalf("expected duplicate visible aliases to collapse to 1 model, got %d", len(models))
	}
	if models[0].ID != "claude-opus-4-6" {
		t.Fatalf("expected visible model id %q, got %q", "claude-opus-4-6", models[0].ID)
	}
	if models[0].OriginalID != "mimo-v2.5-pro" {
		t.Fatalf("expected original id %q, got %q", "mimo-v2.5-pro", models[0].OriginalID)
	}
}
