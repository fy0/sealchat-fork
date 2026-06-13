package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	aiService "sealchat/service/ai"
	"sealchat/utils"
)

func AICapabilitiesGet(ctx *fiber.Ctx) error {
	user := getCurUser(ctx)
	if user == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	if appConfig == nil {
		return ctx.JSON(fiber.Map{"features": []aiService.FeatureCapability{}})
	}
	worldID := strings.TrimSpace(ctx.Query("worldId"))
	features := aiService.AvailableFeatures(appConfig.AI, user.ID, worldID)
	return ctx.JSON(fiber.Map{
		"features": features,
	})
}

func AITaskRun(ctx *fiber.Ctx) error {
	user := getCurUser(ctx)
	if user == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	var body struct {
		WorldID   string `json:"worldId"`
		ChannelID string `json:"channelId"`
		Input     string `json:"input"`
		Source    string `json:"source"`
	}
	if err := ctx.BodyParser(&body); err != nil {
		return err
	}
	runner := aiService.NewRunner(func() *utils.AppConfig { return appConfig }, nil)
	result, err := runner.Run(ctx.Context(), aiService.RunRequest{
		FeatureKey: strings.TrimSpace(ctx.Params("featureKey")),
		UserID:     user.ID,
		WorldID:    strings.TrimSpace(body.WorldID),
		Input:      body.Input,
		Source:     body.Source,
	})
	if err != nil {
		status := fiber.StatusBadRequest
		if strings.Contains(err.Error(), "no ai provider available") {
			status = fiber.StatusServiceUnavailable
		} else if strings.Contains(err.Error(), "unavailable") {
			status = fiber.StatusForbidden
		}
		return ctx.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	return ctx.JSON(fiber.Map{
		"featureKey": result.FeatureKey,
		"result":     result.Result,
		"model":      result.Model,
		"providerId": result.ProviderID,
	})
}
