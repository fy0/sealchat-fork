package utils

import (
	"strings"
	"testing"
)

func TestNormalizeUITextReplaceConfigDefaults(t *testing.T) {
	cfg := NormalizeUITextReplaceConfig(UITextReplaceConfig{})

	if cfg.Enabled {
		t.Fatalf("expected default ui text replace to stay disabled")
	}
	if len(cfg.Rules) != len(DefaultUITextReplaceRules()) {
		t.Fatalf("unexpected default rule count: %d", len(cfg.Rules))
	}
	expectedDefaults := []string{"世界大厅", "世界管理", "术语管理", "公告"}
	for idx, rule := range cfg.Rules {
		if strings.TrimSpace(rule.ID) == "" {
			t.Fatalf("default rule %d should have id", idx)
		}
		if strings.TrimSpace(rule.SearchText) == "" {
			t.Fatalf("default rule %d should have search text", idx)
		}
		if rule.SearchText != expectedDefaults[idx] {
			t.Fatalf("unexpected default search text at %d: %q", idx, rule.SearchText)
		}
	}
}

func TestNormalizeUITextReplaceConfigKeepsValidRules(t *testing.T) {
	cfg := NormalizeUITextReplaceConfig(UITextReplaceConfig{
		Enabled: true,
		Rules: []UITextReplaceRule{
			{
				ID:          "custom-1",
				SearchText:  " 频道 ",
				ReplaceText: " 分区 ",
				Enabled:     true,
			},
			{
				ID:          "",
				SearchText:  " ",
				ReplaceText: "不会保留",
				Enabled:     true,
			},
		},
	})

	if !cfg.Enabled {
		t.Fatalf("expected custom config to stay enabled")
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("expected invalid rules to be dropped, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].SearchText != "频道" {
		t.Fatalf("unexpected normalized search text: %q", cfg.Rules[0].SearchText)
	}
	if cfg.Rules[0].ReplaceText != "分区" {
		t.Fatalf("unexpected normalized replace text: %q", cfg.Rules[0].ReplaceText)
	}
}

func TestValidateUITextReplaceConfigRejectsOversizedInput(t *testing.T) {
	cfg := NormalizeUITextReplaceConfig(UITextReplaceConfig{
		Enabled: true,
		Rules: []UITextReplaceRule{
			{
				ID:          "oversized",
				SearchText:  strings.Repeat("字", MaxUITextReplaceTextLength+1),
				ReplaceText: "x",
				Enabled:     true,
			},
		},
	})

	if err := ValidateUITextReplaceConfig(cfg); err == nil {
		t.Fatalf("expected oversized ui text replace rule to be rejected")
	}
}
