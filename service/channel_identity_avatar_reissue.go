package service

import (
	"fmt"
	"os"
	"strings"

	"gorm.io/gorm"

	"sealchat/model"
	"sealchat/utils"
)

type ChannelIdentityAvatarReissueResult struct {
	ChannelID              string                                `json:"channelId"`
	ProcessedUserCount     int                                   `json:"processedUserCount"`
	ProcessedIdentityCount int                                   `json:"processedIdentityCount"`
	ProcessedVariantCount  int                                   `json:"processedVariantCount"`
	RefreshedIdentityCount int                                   `json:"refreshedIdentityCount"`
	RefreshedVariantCount  int                                   `json:"refreshedVariantCount"`
	CreatedAttachmentCount int                                   `json:"createdAttachmentCount"`
	FailedCount            int                                   `json:"failedCount"`
	Failed                 []ChannelIdentityAvatarReissueFailure `json:"failed"`
	AffectedUserIDs        []string                              `json:"-"`
}

type ChannelIdentityAvatarReissueFailure struct {
	Scope        string `json:"scope"`
	ReferenceID  string `json:"referenceId"`
	TargetUserID string `json:"targetUserId"`
	SourceToken  string `json:"sourceToken,omitempty"`
	Reason       string `json:"reason"`
}

type channelIdentityAvatarReissueIdentityOp struct {
	identityID string
	targetID   string
	source     *model.AttachmentModel
	token      string
}

type channelIdentityAvatarReissueVariantOp struct {
	variantID string
	targetID  string
	source    *model.AttachmentModel
	variant   *model.ChannelIdentityVariantModel
	token     string
}

func ReissueChannelIdentityAvatars(channelID string, operatorUserID string) (*ChannelIdentityAvatarReissueResult, error) {
	channelID = strings.TrimSpace(channelID)
	operatorUserID = strings.TrimSpace(operatorUserID)
	if channelID == "" {
		return nil, ErrChannelNotFound
	}
	if operatorUserID == "" {
		return nil, ErrChannelPermissionDenied
	}

	targetUserIDs, err := listAllManageableChannelIdentityUserIDs(channelID, operatorUserID)
	if err != nil {
		return nil, err
	}

	result := &ChannelIdentityAvatarReissueResult{
		ChannelID:          channelID,
		ProcessedUserCount: len(targetUserIDs),
		Failed:             make([]ChannelIdentityAvatarReissueFailure, 0, 8),
	}
	affected := make(map[string]struct{}, len(targetUserIDs))
	identityOps := make([]channelIdentityAvatarReissueIdentityOp, 0, 16)
	variantOps := make([]channelIdentityAvatarReissueVariantOp, 0, 16)

	for _, targetUserID := range targetUserIDs {
		identities, err := model.ChannelIdentityList(channelID, targetUserID)
		if err != nil {
			return nil, err
		}
		variants, err := model.ChannelIdentityVariantList(channelID, targetUserID)
		if err != nil {
			return nil, err
		}

		result.ProcessedIdentityCount += len(identities)
		result.ProcessedVariantCount += len(variants)

		for _, identity := range identities {
			token := ""
			if identity != nil {
				token = strings.TrimSpace(identity.AvatarAttachmentID)
			}
			if identity == nil || token == "" {
				continue
			}
			source, err := ResolveAttachmentAccessible(targetUserID, operatorUserID, channelID, token)
			if err != nil {
				result.appendFailure("identity", strings.TrimSpace(identity.ID), targetUserID, token, err)
				continue
			}
			identityOps = append(identityOps, channelIdentityAvatarReissueIdentityOp{
				identityID: identity.ID,
				targetID:   targetUserID,
				source:     source,
				token:      token,
			})
		}

		for _, variant := range variants {
			token := ""
			if variant != nil {
				token = strings.TrimSpace(variant.AvatarAttachmentID)
			}
			if variant == nil || token == "" {
				continue
			}
			source, err := ResolveAttachmentAccessible(targetUserID, operatorUserID, channelID, token)
			if err != nil {
				result.appendFailure("variant", strings.TrimSpace(variant.ID), targetUserID, token, err)
				continue
			}
			variantOps = append(variantOps, channelIdentityAvatarReissueVariantOp{
				variantID: variant.ID,
				targetID:  targetUserID,
				source:    source,
				variant:   variant,
				token:     token,
			})
		}
	}

	for _, op := range identityOps {
		if err := reissueIdentityAvatarOp(op, channelID); err != nil {
			result.appendFailure("identity", op.identityID, op.targetID, op.token, err)
			continue
		}
		result.RefreshedIdentityCount++
		result.CreatedAttachmentCount++
		affected[op.targetID] = struct{}{}
	}

	for _, op := range variantOps {
		if err := reissueVariantAvatarOp(op, channelID); err != nil {
			result.appendFailure("variant", op.variantID, op.targetID, op.token, err)
			continue
		}
		result.RefreshedVariantCount++
		result.CreatedAttachmentCount++
		affected[op.targetID] = struct{}{}
	}

	result.AffectedUserIDs = make([]string, 0, len(affected))
	for _, targetUserID := range targetUserIDs {
		if _, ok := affected[targetUserID]; ok {
			result.AffectedUserIDs = append(result.AffectedUserIDs, targetUserID)
		}
	}
	return result, nil
}

func (r *ChannelIdentityAvatarReissueResult) appendFailure(scope string, referenceID string, targetUserID string, sourceToken string, err error) {
	if r == nil || err == nil {
		return
	}
	r.Failed = append(r.Failed, ChannelIdentityAvatarReissueFailure{
		Scope:        strings.TrimSpace(scope),
		ReferenceID:  strings.TrimSpace(referenceID),
		TargetUserID: strings.TrimSpace(targetUserID),
		SourceToken:  strings.TrimSpace(sourceToken),
		Reason:       err.Error(),
	})
	r.FailedCount = len(r.Failed)
}

func listAllManageableChannelIdentityUserIDs(channelID string, operatorUserID string) ([]string, error) {
	page := 1
	result := make([]string, 0, 8)
	for {
		items, err := ListChannelIdentityManageCandidates(ChannelIdentityManageCandidateQuery{
			ChannelID: channelID,
			ActorID:   operatorUserID,
			Page:      page,
			PageSize:  100,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range items.Items {
			if item == nil {
				continue
			}
			targetUserID := strings.TrimSpace(item.UserID)
			if targetUserID == "" {
				continue
			}
			if _, err := ResolveChannelIdentityActor(channelID, operatorUserID, targetUserID); err != nil {
				continue
			}
			result = append(result, targetUserID)
		}
		if len(items.Items) == 0 || int64(page*items.PageSize) >= items.Total {
			break
		}
		page++
	}
	return dedupeStrings(result), nil
}

func createReissuedAttachmentTokenTx(tx *gorm.DB, source *model.AttachmentModel, targetUserID string, channelID string) (string, error) {
	tempPath, err := MaterializeAttachmentToTempFile(source)
	if err != nil {
		return "", err
	}
	defer os.Remove(tempPath)

	location, err := PersistAttachmentFileForceNew(source.Hash, source.Size, tempPath, source.MimeType, source.Filename)
	if err != nil {
		return "", err
	}

	item := &model.AttachmentModel{
		StringPKBaseModel: model.StringPKBaseModel{ID: utils.NewID()},
		Hash:              append(model.ByteArray(nil), source.Hash...),
		Filename:          source.Filename,
		Size:              source.Size,
		MimeType:          source.MimeType,
		IsAnimated:        source.IsAnimated,
		UserID:            strings.TrimSpace(targetUserID),
		ChannelID:         strings.TrimSpace(channelID),
		StorageType:       location.StorageType,
		ObjectKey:         location.ObjectKey,
		ExternalURL:       location.ExternalURL,
		Extra:             source.Extra,
		Note:              source.Note,
		RootID:            source.RootID,
		RootIDType:        source.RootIDType,
		ParentID:          source.ParentID,
		ParentIDType:      source.ParentIDType,
		IsTemp:            false,
		CreatorName:       source.CreatorName,
		CreatorAvatar:     source.CreatorAvatar,
	}
	if err := tx.Create(item).Error; err != nil {
		return "", err
	}
	return "id:" + item.ID, nil
}

func reissueIdentityAvatarOp(op channelIdentityAvatarReissueIdentityOp, channelID string) error {
	return model.GetDB().Transaction(func(tx *gorm.DB) error {
		token, err := createReissuedAttachmentTokenTx(tx, op.source, op.targetID, channelID)
		if err != nil {
			return fmt.Errorf("重签发频道角色头像失败(identity=%s): %w", strings.TrimSpace(op.identityID), err)
		}
		if err := tx.Model(&model.ChannelIdentityModel{}).
			Where("id = ?", op.identityID).
			Update("avatar_attachment_id", token).Error; err != nil {
			return err
		}
		return nil
	})
}

func reissueVariantAvatarOp(op channelIdentityAvatarReissueVariantOp, channelID string) error {
	return model.GetDB().Transaction(func(tx *gorm.DB) error {
		token, err := createReissuedAttachmentTokenTx(tx, op.source, op.targetID, channelID)
		if err != nil {
			return fmt.Errorf("重签发头像差分失败(variant=%s): %w", strings.TrimSpace(op.variantID), err)
		}
		appearanceJSON, err := rebuildVariantAvatarAppearanceJSON(op.variant, token)
		if err != nil {
			return fmt.Errorf("更新头像差分 appearance 失败(variant=%s): %w", strings.TrimSpace(op.variantID), err)
		}
		if err := tx.Model(&model.ChannelIdentityVariantModel{}).
			Where("id = ?", op.variantID).
			Updates(map[string]any{
				"avatar_attachment_id": token,
				"appearance_json":      appearanceJSON,
			}).Error; err != nil {
			return err
		}
		return nil
	})
}

func rebuildVariantAvatarAppearanceJSON(variant *model.ChannelIdentityVariantModel, token string) (string, error) {
	appearance := map[string]any{}
	if variant != nil {
		appearance = variant.Appearance()
	}
	appearance["avatarAttachmentId"] = strings.TrimSpace(token)
	return serializeChannelIdentityVariantAppearanceJSON(appearance)
}
