package api

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	aiService "sealchat/service/ai"
	"sealchat/utils"
)

func AdminAIConfigGet(ctx *fiber.Ctx) error {
	cfg := sanitizeConfigForAdmin(appConfig).AI
	return ctx.JSON(fiber.Map{
		"config": cfg,
	})
}

func AdminAIConfigUpdate(ctx *fiber.Ctx) error {
	var body struct {
		Config utils.AIConfig `json:"config"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return err
	}
	current := appConfig
	if current == nil {
		current = &utils.AppConfig{}
	}
	incoming := *current
	incoming.AI = body.Config
	merged := mergeConfigForWrite(current, &incoming)
	merged.AI = utils.NormalizeAIConfig(merged.AI)
	if err := utils.ValidateAIConfig(merged.AI); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}
	appConfig = merged
	utils.WriteConfig(appConfig)
	SyncConfigToDB(appConfig, "api")
	return ctx.JSON(fiber.Map{
		"config": sanitizeConfigForAdmin(appConfig).AI,
	})
}

func AdminAIProviderTest(ctx *fiber.Ctx) error {
	var body struct {
		ProviderID string `json:"providerId"`
		Model      string `json:"model"`
		Prompt     string `json:"prompt"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return err
	}
	if appConfig == nil {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "AI 配置不可用"})
	}
	cfg := utils.NormalizeAIConfig(appConfig.AI)
	for _, provider := range cfg.Providers {
		if provider.ID != strings.TrimSpace(body.ProviderID) {
			continue
		}
		model := strings.TrimSpace(body.Model)
		if model == "" && len(provider.Models) > 0 {
			model = provider.Models[0]
		}
		client := aiService.NewRunner(func() *utils.AppConfig {
			return &utils.AppConfig{AI: utils.AIConfig{
				Enabled:   true,
				Providers: []utils.AIProviderConfig{provider},
				Features: map[string]utils.AIFeatureConfig{
					aiService.FeaturePolish: {
						Enabled:       true,
						DefaultPrompt: "你是连通性测试助手。按原样返回用户输入。",
						DefaultModel:  model,
						Access: utils.AIFeatureAccessConfig{
							Mode: utils.AIFeatureAccessAll,
						},
					},
				},
			}}
		}, nil)
		result, err := client.Run(context.Background(), aiService.RunRequest{
			FeatureKey: aiService.FeaturePolish,
			UserID:     "admin-test",
			WorldID:    "",
			Input:      strings.TrimSpace(body.Prompt),
			Source:     "platform",
		})
		if err != nil {
			return ctx.Status(fiber.StatusBadGateway).JSON(fiber.Map{"message": err.Error()})
		}
		return ctx.JSON(fiber.Map{
			"providerId": result.ProviderID,
			"model":      result.Model,
			"result":     result.Result,
		})
	}
	return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"message": "AI provider 不存在"})
}
