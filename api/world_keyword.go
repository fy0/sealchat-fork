package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"sealchat/protocol"
	"sealchat/service"
	"sealchat/utils"
)

func WorldKeywordListHandler(c *fiber.Ctx) error {
	user := getCurUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	worldID := c.Params("worldId")
	page := parseQueryIntDefault(c, "page", 1)
	pageSize := parseQueryIntDefault(c, "pageSize", 50)
	query := strings.TrimSpace(c.Query("q"))
	includeDisabled := c.QueryBool("includeDisabled")
	items, total, err := service.WorldKeywordList(worldID, user.ID, service.WorldKeywordListOptions{
		Page:            page,
		PageSize:        pageSize,
		Query:           query,
		IncludeDisabled: includeDisabled,
	})
	if err != nil {
		status := fiber.StatusInternalServerError
		switch err {
		case service.ErrWorldPermission:
			status = fiber.StatusForbidden
		case service.ErrWorldNotFound:
			status = fiber.StatusNotFound
		default:
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func WorldKeywordCreateHandler(c *fiber.Ctx) error {
	user := getCurUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	worldID := c.Params("worldId")
	var payload service.WorldKeywordInput
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "参数错误"})
	}
	item, err := service.WorldKeywordCreate(worldID, user.ID, payload)
	if err != nil {
		status := fiber.StatusBadRequest
		switch err {
		case service.ErrWorldPermission:
			status = fiber.StatusForbidden
		case service.ErrWorldNotFound:
			status = fiber.StatusNotFound
		default:
			if strings.Contains(err.Error(), "关键词") {
				status = fiber.StatusBadRequest
			} else {
				status = fiber.StatusInternalServerError
			}
		}
		return c.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	broadcastWorldKeywordEvent(worldID, []string{item.ID}, "created")
	return c.Status(http.StatusCreated).JSON(fiber.Map{"item": item})
}

func WorldKeywordUpdateHandler(c *fiber.Ctx) error {
	user := getCurUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	worldID := c.Params("worldId")
	keywordID := c.Params("keywordId")
	var payload service.WorldKeywordInput
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "参数错误"})
	}
	item, err := service.WorldKeywordUpdate(worldID, keywordID, user.ID, payload)
	if err != nil {
		status := fiber.StatusInternalServerError
		switch err {
		case service.ErrWorldPermission:
			status = fiber.StatusForbidden
		case service.ErrWorldKeywordNotFound:
			status = fiber.StatusNotFound
		default:
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	broadcastWorldKeywordEvent(worldID, []string{item.ID}, "updated")
	return c.JSON(fiber.Map{"item": item})
}

func WorldKeywordDeleteHandler(c *fiber.Ctx) error {
	user := getCurUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	worldID := c.Params("worldId")
	keywordID := c.Params("keywordId")
	if err := service.WorldKeywordDelete(worldID, keywordID, user.ID); err != nil {
		status := fiber.StatusInternalServerError
		switch err {
		case service.ErrWorldPermission:
			status = fiber.StatusForbidden
		case service.ErrWorldKeywordNotFound:
			status = fiber.StatusNotFound
		default:
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	broadcastWorldKeywordEvent(worldID, []string{keywordID}, "deleted")
	return c.SendStatus(fiber.StatusNoContent)
}

func WorldKeywordBulkDeleteHandler(c *fiber.Ctx) error {
	user := getCurUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	worldID := c.Params("worldId")
	var payload struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "参数错误"})
	}
	count, err := service.WorldKeywordBulkDelete(worldID, payload.IDs, user.ID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err == service.ErrWorldPermission {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	if count > 0 {
		broadcastWorldKeywordEvent(worldID, payload.IDs, "deleted")
	}
	return c.JSON(fiber.Map{"deleted": count})
}

func WorldKeywordImportHandler(c *fiber.Ctx) error {
	user := getCurUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	worldID := c.Params("worldId")
	var payload struct {
		Items   []service.WorldKeywordInput `json:"items"`
		Replace bool                        `json:"replace"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "参数错误"})
	}
	stats, err := service.WorldKeywordImport(worldID, user.ID, payload.Items, payload.Replace)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err == service.ErrWorldPermission {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	broadcastWorldKeywordEvent(worldID, nil, "imported")
	return c.JSON(fiber.Map{"stats": stats})
}

func WorldKeywordExportHandler(c *fiber.Ctx) error {
	user := getCurUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "未登录"})
	}
	worldID := c.Params("worldId")
	items, err := service.WorldKeywordExport(worldID, user.ID)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err == service.ErrWorldPermission {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{"items": items})
}

func broadcastWorldKeywordEvent(worldID string, keywordIDs []string, operation string) {
	if strings.TrimSpace(worldID) == "" {
		return
	}
	payload := map[string]interface{}{
		"worldId":    worldID,
		"keywordIds": keywordIDs,
		"operation":  operation,
		"version":    time.Now().UnixMilli(),
	}
	event := &protocol.Event{
		Type: protocol.EventWorldKeywordsUpdated,
		Argv: &protocol.Argv{Options: payload},
	}
	broadcastEventToWorld(worldID, event)
}

func broadcastEventToWorld(worldID string, event *protocol.Event) {
	if userId2ConnInfoGlobal == nil {
		return
	}
	event.Timestamp = time.Now().Unix()
	userId2ConnInfoGlobal.Range(func(_ string, conns *utils.SyncMap[*WsSyncConn, *ConnInfo]) bool {
		conns.Range(func(conn *WsSyncConn, info *ConnInfo) bool {
			if info != nil && info.WorldId == worldID {
				_ = conn.WriteJSON(struct {
					protocol.Event
					Op protocol.Opcode `json:"op"`
				}{
					Event: *event,
					Op:    protocol.OpEvent,
				})
			}
			return true
		})
		return true
	})
}
