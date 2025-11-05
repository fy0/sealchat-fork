package api

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"sealchat/model"
	"sealchat/service"
	"sealchat/utils"
)

func UserEmojiAdd(c *fiber.Ctx) error {
	ui := getCurUser(c)

	var body struct {
		AttachmentId string `json:"attachmentId"`
		Remark       string `json:"remark"`
	}
	if err := c.BodyParser(&body); err != nil {
		return wrapErrorStatus(c, fiber.StatusBadRequest, err, "请求参数错误")
	}
	if strings.TrimSpace(body.AttachmentId) == "" {
		return wrapErrorStatus(c, fiber.StatusBadRequest, nil, "附件ID不能为空")
	}

	remark := strings.TrimSpace(body.Remark)
	if remark != "" && !service.GalleryValidateRemark(remark) {
		return wrapErrorStatus(c, fiber.StatusBadRequest, nil, service.ErrGalleryRemarkInvalid.Error())
	}

	item, err := model.UserEmojiCreate(ui.ID, body.AttachmentId, remark)
	if err != nil {
		return wrapError(c, err, "收藏表情失败")
	}
	return c.JSON(fiber.Map{
		"item": item,
	})
}

func UserEmojiDelete(c *fiber.Ctx) error {
	db := model.GetDB()
	ui := getCurUser(c)
	var reqBody struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&reqBody); err != nil {
		return wrapErrorStatus(c, fiber.StatusBadRequest, err, "无效的请求参数")
	}
	ids := reqBody.IDs
	if len(ids) == 0 {
		return wrapErrorStatus(c, fiber.StatusBadRequest, nil, "ID列表不能为空")
	}
	result := db.Unscoped().Where("user_id = ?", ui.ID).Delete(&model.UserEmojiModel{}, "id IN ?", ids)
	if result.Error != nil {
		return wrapError(c, result.Error, "删除表情失败")
	}
	return c.JSON(fiber.Map{
		"message": "表情删除成功",
		"count":   result.RowsAffected,
	})
}

func UserEmojiList(c *fiber.Ctx) error {
	ui := getCurUser(c)

	return utils.APIPaginatedList(c, func(page, pageSize int) ([]*model.UserEmojiModel, int64, error) {
		return model.UserEmojiList(ui.ID, 1, -1)
	})
}

func UserEmojiUpdate(c *fiber.Ctx) error {
	emojiID := c.Params("id")
	if emojiID == "" {
		return wrapErrorStatus(c, fiber.StatusBadRequest, nil, "表情ID不能为空")
	}
	var payload struct {
		Remark string `json:"remark"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return wrapErrorStatus(c, fiber.StatusBadRequest, err, "请求参数错误")
	}

	item, err := model.GetUserEmoji(emojiID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return wrapErrorStatus(c, fiber.StatusNotFound, err, "表情不存在")
		}
		return wrapError(c, err, "读取表情信息失败")
	}

	ui := getCurUser(c)
	if item.UserID != ui.ID {
		return wrapErrorStatus(c, fiber.StatusForbidden, nil, "无法编辑他人表情")
	}

	remark := strings.TrimSpace(payload.Remark)
	if remark != "" && !service.GalleryValidateRemark(remark) {
		return wrapErrorStatus(c, fiber.StatusBadRequest, nil, service.ErrGalleryRemarkInvalid.Error())
	}

	if err := model.UpdateUserEmoji(item, map[string]interface{}{"remark": remark}); err != nil {
		return wrapError(c, err, "更新表情备注失败")
	}
	item.Remark = remark
	return c.JSON(fiber.Map{"item": item})
}
