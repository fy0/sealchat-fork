package utils

import (
	"fmt"
	"strings"
)

const (
	MaxUITextReplaceRuleCount  = 100
	MaxUITextReplaceTextLength = 64
)

type UITextReplaceRule struct {
	ID          string `json:"id" yaml:"id"`
	SearchText  string `json:"searchText" yaml:"searchText"`
	ReplaceText string `json:"replaceText" yaml:"replaceText"`
	Enabled     bool   `json:"enabled" yaml:"enabled"`
}

type UITextReplaceConfig struct {
	Enabled bool                `json:"enabled" yaml:"enabled"`
	Rules   []UITextReplaceRule `json:"rules" yaml:"rules"`
}

func DefaultUITextReplaceRules() []UITextReplaceRule {
	return []UITextReplaceRule{
		{ID: "default-world-lobby", SearchText: "世界大厅", ReplaceText: "世界大厅", Enabled: true},
		{ID: "default-world-manage", SearchText: "世界管理", ReplaceText: "世界管理", Enabled: true},
		{ID: "default-glossary-manage", SearchText: "术语管理", ReplaceText: "术语管理", Enabled: true},
		{ID: "default-announcement", SearchText: "公告", ReplaceText: "公告", Enabled: true},
	}
}

func NormalizeUITextReplaceConfig(cfg UITextReplaceConfig) UITextReplaceConfig {
	result := UITextReplaceConfig{
		Enabled: cfg.Enabled,
		Rules:   make([]UITextReplaceRule, 0, len(cfg.Rules)),
	}
	source := cfg.Rules
	if len(source) == 0 {
		source = DefaultUITextReplaceRules()
	}
	for idx, item := range source {
		searchText := strings.TrimSpace(item.SearchText)
		if searchText == "" {
			continue
		}
		ruleID := strings.TrimSpace(item.ID)
		if ruleID == "" {
			ruleID = fmt.Sprintf("ui-text-replace-%d", idx+1)
		}
		result.Rules = append(result.Rules, UITextReplaceRule{
			ID:          ruleID,
			SearchText:  searchText,
			ReplaceText: strings.TrimSpace(item.ReplaceText),
			Enabled:     item.Enabled,
		})
	}
	if len(result.Rules) == 0 {
		result.Rules = DefaultUITextReplaceRules()
	}
	return result
}

func ValidateUITextReplaceConfig(cfg UITextReplaceConfig) error {
	if len(cfg.Rules) > MaxUITextReplaceRuleCount {
		return fmt.Errorf("界面文本替换规则不能超过 %d 条", MaxUITextReplaceRuleCount)
	}
	seenIDs := make(map[string]struct{}, len(cfg.Rules))
	for _, item := range cfg.Rules {
		ruleID := strings.TrimSpace(item.ID)
		searchText := strings.TrimSpace(item.SearchText)
		replaceText := strings.TrimSpace(item.ReplaceText)
		if ruleID == "" {
			return fmt.Errorf("界面文本替换规则 id 不能为空")
		}
		if searchText == "" {
			return fmt.Errorf("界面文本替换原文不能为空")
		}
		if _, exists := seenIDs[ruleID]; exists {
			return fmt.Errorf("界面文本替换规则 id 重复: %s", ruleID)
		}
		seenIDs[ruleID] = struct{}{}
		if len([]rune(searchText)) > MaxUITextReplaceTextLength {
			return fmt.Errorf("界面文本替换原文不能超过 %d 个字符", MaxUITextReplaceTextLength)
		}
		if len([]rune(replaceText)) > MaxUITextReplaceTextLength {
			return fmt.Errorf("界面文本替换目标文案不能超过 %d 个字符", MaxUITextReplaceTextLength)
		}
	}
	return nil
}
