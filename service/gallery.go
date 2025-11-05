package service

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"gorm.io/gorm"

	"sealchat/model"
	"sealchat/pm"
)

var (
	galleryRemarkPattern    = regexp.MustCompile(`^[\p{L}\p{N}_]{1,64}$`)
	ErrGalleryRemarkInvalid = errors.New("备注仅支持字母、数字和下划线，长度不超过64")
	ErrGalleryPermission    = errors.New("缺少快捷表情资源操作权限")
	ErrGalleryQuotaExceeded = errors.New("快捷表情容量不足")
)

const defaultCollectionName = "默认分类"

func GalleryValidateRemark(remark string) bool {
	if remark == "" {
		return false
	}
	return galleryRemarkPattern.MatchString(remark)
}

func GalleryEnsureDefaultCollection(ownerType model.OwnerType, ownerID, creatorID string) (*model.GalleryCollection, error) {
	db := model.GetDB()
	var col model.GalleryCollection
	err := db.Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		Order("`order`, created_at").
		Limit(1).
		Take(&col).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		col = model.GalleryCollection{
			OwnerType: ownerType,
			OwnerID:   ownerID,
			Name:      defaultCollectionName,
			Order:     0,
			CreatedBy: creatorID,
			UpdatedBy: creatorID,
		}
		col.StringPKBaseModel.Init()
		if err = db.Create(&col).Error; err != nil {
			return nil, err
		}
		return &col, nil
	}
	if err != nil {
		return nil, err
	}
	return &col, nil
}

func GalleryListCollections(ownerType model.OwnerType, ownerID, creatorID string) ([]*model.GalleryCollection, error) {
	cols, err := model.ListGalleryCollections(ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		col, err := GalleryEnsureDefaultCollection(ownerType, ownerID, creatorID)
		if err != nil {
			return nil, err
		}
		cols = []*model.GalleryCollection{col}
	}
	return cols, nil
}

func GalleryEnsureCanRead(userID string, ownerType model.OwnerType, ownerID string) bool {
	switch ownerType {
	case model.OwnerTypeUser:
		return ownerID == userID
	case model.OwnerTypeChannel:
		return pm.Can(userID, ownerID, pm.PermFuncChannelRead)
	default:
		return false
	}
}

func GalleryEnsureCanManage(userID string, ownerType model.OwnerType, ownerID string) bool {
	switch ownerType {
	case model.OwnerTypeUser:
		return ownerID == userID
	case model.OwnerTypeChannel:
		return pm.CanWithChannelRole(userID, ownerID, pm.PermFuncChannelManageGallery)
	default:
		return false
	}
}

func GalleryUserUsageBytes(userID string) (int64, error) {
	var total int64
	err := model.GetDB().Model(&model.GalleryItem{}).
		Where("created_by = ?", userID).
		Select("COALESCE(SUM(size),0)").
		Scan(&total).Error
	return total, err
}

func GalleryEnsureQuota(userID string, additional int64, limitBytes int64) error {
	used, err := GalleryUserUsageBytes(userID)
	if err != nil {
		return err
	}
	if used+additional > limitBytes {
		return ErrGalleryQuotaExceeded
	}
	return nil
}

func GalleryUpdateCollectionQuota(collectionID string) error {
	var total int64
	db := model.GetDB()
	if err := db.Model(&model.GalleryItem{}).
		Where("collection_id = ?", collectionID).
		Select("COALESCE(SUM(size),0)").
		Scan(&total).Error; err != nil {
		return err
	}

	return db.Model(&model.GalleryCollection{}).
		Where("id = ?", collectionID).
		Update("quota_used", total).Error
}

func GalleryBatchUpdateCollectionQuota(collectionIDs []string) error {
	if len(collectionIDs) == 0 {
		return nil
	}
	unique := map[string]struct{}{}
	for _, id := range collectionIDs {
		if id == "" {
			continue
		}
		if _, ok := unique[id]; ok {
			continue
		}
		unique[id] = struct{}{}
		if err := GalleryUpdateCollectionQuota(id); err != nil {
			return err
		}
	}
	return nil
}

func GalleryThumbFilename(itemID string, ext string) string {
	return filepath.Join("./data/gallery/thumbs", itemID+ext)
}

func NormalizeRemark(input, filename string) string {
	remark := strings.TrimSpace(input)
	if remark == "" {
		remark = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	var builder strings.Builder
	builder.Grow(len(remark))
	lastUnderscore := false
	for _, r := range remark {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastUnderscore = false
		case r == '_':
			if !lastUnderscore {
				builder.WriteRune('_')
				lastUnderscore = true
			}
		default:
			if !lastUnderscore {
				builder.WriteRune('_')
				lastUnderscore = true
			}
		}
	}

	out := strings.Trim(builder.String(), "_")
	if out == "" {
		out = "img"
	}
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
