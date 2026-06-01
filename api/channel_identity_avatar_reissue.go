package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"sealchat/service"
)

func ChannelIdentityAvatarReissue(c *fiber.Ctx) error {
	channelID := strings.TrimSpace(c.Params("channelId"))
	ctx, err := resolveChannelIdentityActorFromRequest(c, channelID, "")
	if err != nil {
		return handleChannelIdentityActorErr(c, err)
	}

	result, err := service.ReissueChannelIdentityAvatars(channelID, ctx.OperatorUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrChannelNotFound),
			errors.Is(err, service.ErrChannelWorldRequired),
			errors.Is(err, service.ErrChannelIdentityTargetNotInChannel),
			errors.Is(err, service.ErrChannelIdentityDelegationDisabled),
			errors.Is(err, service.ErrChannelIdentityDelegationForbidden),
			errors.Is(err, service.ErrChannelPermissionDenied):
			return handleChannelIdentityActorErr(c, err)
		default:
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	for _, targetUserID := range result.AffectedUserIDs {
		broadcastChannelIdentityRefresh(channelIdentityRefreshPayload{
			ChannelID:      channelID,
			TargetUserID:   targetUserID,
			OperatorUserID: ctx.OperatorUserID,
			Reason:         "identity-avatar-reissue",
		})
	}

	return c.JSON(result)
}
