package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"sealchat/model"
	"sealchat/utils"
)

var (
	ErrWorldNotFound       = errors.New("world not found")
	ErrWorldPermission     = errors.New("world permission denied")
	ErrWorldInviteInvalid  = errors.New("world invite invalid")
	ErrWorldMemberInvalid  = errors.New("world member invalid")
	ErrWorldOwnerImmutable = errors.New("world owner immutable")
)

type WorldCreateParams struct {
	Name        string
	Description string
	Visibility  string
	Avatar      string
}

type WorldUpdateParams struct {
	Name              string
	Description       string
	Visibility        string
	Avatar            string
	EnforceMembership *bool
}

func GetOrCreateDefaultWorld() (*model.WorldModel, error) {
	db := model.GetDB()
	var world model.WorldModel
	if err := db.Where("status = ?", "active").Order("created_at asc").Limit(1).Find(&world).Error; err != nil {
		return nil, err
	}
	if world.ID != "" {
		return &world, nil
	}
	w := &model.WorldModel{
		Name:        "公共世界",
		Description: "系统自动创建的默认世界",
		Visibility:  model.WorldVisibilityPublic,
		Status:      "active",
	}
	if err := db.Create(w).Error; err != nil {
		return nil, err
	}
	return w, nil
}

func GetWorldByID(worldID string) (*model.WorldModel, error) {
	if strings.TrimSpace(worldID) == "" {
		return nil, ErrWorldNotFound
	}
	var world model.WorldModel
	if err := model.GetDB().Where("id = ?", worldID).Limit(1).Find(&world).Error; err != nil {
		return nil, err
	}
	if world.ID == "" {
		return nil, ErrWorldNotFound
	}
	return &world, nil
}

func WorldCreate(ownerID string, params WorldCreateParams) (*model.WorldModel, *model.ChannelModel, error) {
	name := strings.TrimSpace(params.Name)
	if name == "" {
		return nil, nil, errors.New("世界名称不能为空")
	}
	visibility := params.Visibility
	if visibility == "" {
		visibility = model.WorldVisibilityPublic
	}
	world := &model.WorldModel{
		Name:              name,
		Description:       params.Description,
		Avatar:            params.Avatar,
		Visibility:        visibility,
		OwnerID:           ownerID,
		EnforceMembership: false,
		Status:            "active",
	}
	db := model.GetDB()
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(world).Error; err != nil {
			return err
		}
		member := &model.WorldMemberModel{
			WorldID:  world.ID,
			UserID:   ownerID,
			Role:     model.WorldRoleOwner,
			JoinedAt: time.Now(),
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	channelName := fmt.Sprintf("%s大厅", name)
	defaultChannel := ChannelNew(utils.NewID(), "public", channelName, world.ID, ownerID, "")
	if defaultChannel != nil {
		_ = db.Model(&model.WorldModel{}).
			Where("id = ?", world.ID).
			Update("default_channel_id", defaultChannel.ID).Error
	}
	return world, defaultChannel, nil
}

func WorldUpdate(worldID, actorID string, params WorldUpdateParams) (*model.WorldModel, error) {
	world := &model.WorldModel{}
	if err := model.GetDB().Where("id = ? AND status = ?", worldID, "active").Limit(1).Find(world).Error; err != nil {
		return nil, err
	}
	if world.ID == "" {
		return nil, ErrWorldNotFound
	}
	if !IsWorldAdmin(worldID, actorID) {
		return nil, ErrWorldPermission
	}
	updates := map[string]interface{}{}
	if name := strings.TrimSpace(params.Name); name != "" {
		updates["name"] = name
	}
	if params.Description != "" {
		updates["description"] = params.Description
	}
	if params.Avatar != "" {
		updates["avatar"] = params.Avatar
	}
	if params.Visibility != "" {
		updates["visibility"] = params.Visibility
	}
	if params.EnforceMembership != nil {
		updates["enforce_membership"] = *params.EnforceMembership
	}
	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := model.GetDB().Model(world).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return world, nil
}

func WorldDelete(worldID, actorID string) error {
	if !IsWorldOwner(worldID, actorID) {
		return ErrWorldPermission
	}
	db := model.GetDB()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.WorldModel{}).
			Where("id = ?", worldID).
			Updates(map[string]any{"status": "archived", "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		if err := tx.Where("world_id = ?", worldID).Delete(&model.WorldMemberModel{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.ChannelModel{}).
			Where("world_id = ?", worldID).
			Updates(map[string]any{"status": "archived", "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.WorldInviteModel{}).
			Where("world_id = ?", worldID).
			Updates(map[string]any{"status": "archived", "updated_at": time.Now()}).Error; err != nil {
			return err
		}
		if err := tx.Where("world_id = ?", worldID).Delete(&model.WorldFavoriteModel{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.MessageModel{}).
			Where("channel_id IN (?)", tx.Table("channels").Select("id").Where("world_id = ?", worldID)).
			Updates(map[string]any{"is_archived": true, "archived_at": time.Now(), "archive_reason": "world_deleted"}).Error; err != nil {
			return err
		}
		return nil
	})
}

func WorldJoin(worldID, userID string) (*model.WorldMemberModel, error) {
	db := model.GetDB()
	var world model.WorldModel
	if err := db.Where("id = ? AND status = ?", worldID, "active").Limit(1).Find(&world).Error; err != nil {
		return nil, err
	}
	if world.ID == "" {
		return nil, ErrWorldNotFound
	}
	member := &model.WorldMemberModel{}
	if err := db.Where("world_id = ? AND user_id = ?", worldID, userID).Limit(1).Find(member).Error; err != nil {
		return nil, err
	}
	if member.ID != "" {
		if err := ensureWorldChannelMemberships(worldID, userID); err != nil {
			return member, err
		}
		return member, nil
	}
	member = &model.WorldMemberModel{
		WorldID:  worldID,
		UserID:   userID,
		Role:     model.WorldRoleMember,
		JoinedAt: time.Now(),
	}
	if err := db.Create(member).Error; err != nil {
		return nil, err
	}
	if err := ensureWorldChannelMemberships(worldID, userID); err != nil {
		return member, err
	}
	return member, nil
}

func WorldLeave(worldID, userID string) error {
	if IsWorldOwner(worldID, userID) {
		return errors.New("世界拥有者无法退出，请先转移所有权或删除世界")
	}
	db := model.GetDB()
	if err := db.Where("world_id = ? AND user_id = ?", worldID, userID).Delete(&model.WorldMemberModel{}).Error; err != nil {
		return err
	}
	_ = db.Where("world_id = ? AND user_id = ?", worldID, userID).Delete(&model.WorldFavoriteModel{})
	return nil
}

func IsWorldOwner(worldID, userID string) bool {
	return worldRoleEquals(worldID, userID, model.WorldRoleOwner)
}

func IsWorldAdmin(worldID, userID string) bool {
	if worldRoleEquals(worldID, userID, model.WorldRoleOwner) {
		return true
	}
	return worldRoleEquals(worldID, userID, model.WorldRoleAdmin)
}

func IsWorldMember(worldID, userID string) bool {
	return worldRoleEquals(worldID, userID, "")
}

func worldRoleEquals(worldID, userID, role string) bool {
	var member model.WorldMemberModel
	err := model.GetDB().Where("world_id = ? AND user_id = ?", worldID, userID).Limit(1).Find(&member).Error
	if err != nil || member.ID == "" {
		return false
	}
	if role == "" {
		return true
	}
	return member.Role == role
}

func ListWorldMembers(worldID string, limit int) ([]*model.WorldMemberModel, error) {
	if limit <= 0 {
		limit = 20
	}
	var members []*model.WorldMemberModel
	err := model.GetDB().Where("world_id = ?", worldID).
		Order("joined_at asc").
		Limit(limit).
		Find(&members).Error
	return members, err
}

type WorldMemberDetail struct {
	ID       string    `json:"id"`
	WorldID  string    `json:"worldId"`
	UserID   string    `json:"userId"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joinedAt"`
	Username string    `json:"username"`
	Nickname string    `json:"nickname"`
}

func ListWorldMembersDetail(worldID string, page, pageSize int, keyword string) ([]*WorldMemberDetail, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	db := model.GetDB()
	query := db.Table("world_members AS wm").
		Select("wm.id, wm.world_id, wm.user_id, wm.role, wm.joined_at, u.username, u.nickname").
		Joins("LEFT JOIN users u ON u.id = wm.user_id").
		Where("wm.world_id = ?", worldID)
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("wm.user_id LIKE ? OR u.username LIKE ? OR u.nickname LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var rows []struct {
		ID       string
		WorldID  string
		UserID   string
		Role     string
		JoinedAt time.Time
		Username string
		Nickname string
	}
	if err := query.Order("wm.joined_at asc").
		Offset(offset).
		Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*WorldMemberDetail, 0, len(rows))
	for _, row := range rows {
		result = append(result, &WorldMemberDetail{
			ID:       row.ID,
			WorldID:  row.WorldID,
			UserID:   row.UserID,
			Role:     row.Role,
			JoinedAt: row.JoinedAt,
			Username: row.Username,
			Nickname: row.Nickname,
		})
	}
	return result, total, nil
}

func ensureWorldChannelMemberships(worldID, userID string) error {
	channels, err := ChannelListByWorld(worldID)
	if err != nil {
		return err
	}
	for _, ch := range channels {
		if ch == nil || strings.TrimSpace(ch.ID) == "" {
			continue
		}
		if _, err := model.MemberGetByUserIDAndChannelIDBase(userID, ch.ID, "", true); err != nil {
			return err
		}
	}
	return nil
}

func ListWorldFavorites(userID string) ([]string, error) {
	return model.ListWorldFavoriteIDs(userID)
}

func ToggleWorldFavorite(worldID, userID string, favorite bool) ([]string, error) {
	worldID = strings.TrimSpace(worldID)
	if worldID == "" {
		return nil, ErrWorldNotFound
	}
	if !IsWorldMember(worldID, userID) {
		return nil, ErrWorldPermission
	}
	if err := model.SetWorldFavorite(worldID, userID, favorite); err != nil {
		return nil, err
	}
	return model.ListWorldFavoriteIDs(userID)
}

func WorldRemoveMember(worldID, actorID, targetUserID string) error {
	if strings.TrimSpace(targetUserID) == "" {
		return ErrWorldMemberInvalid
	}
	if !IsWorldAdmin(worldID, actorID) {
		return ErrWorldPermission
	}
	if IsWorldOwner(worldID, targetUserID) {
		return ErrWorldOwnerImmutable
	}
	return WorldLeave(worldID, targetUserID)
}

func WorldUpdateMemberRole(worldID, actorID, targetUserID, role string) error {
	role = strings.TrimSpace(role)
	if role != model.WorldRoleAdmin && role != model.WorldRoleMember {
		return ErrWorldMemberInvalid
	}
	if !IsWorldAdmin(worldID, actorID) {
		return ErrWorldPermission
	}
	if IsWorldOwner(worldID, targetUserID) {
		return ErrWorldOwnerImmutable
	}
	db := model.GetDB()
	res := db.Model(&model.WorldMemberModel{}).
		Where("world_id = ? AND user_id = ?", worldID, targetUserID).
		Updates(map[string]any{"role": role, "updated_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrWorldMemberInvalid
	}
	return nil
}

func WorldInviteCreate(worldID, creatorID string, ttlMinutes int, maxUse int, memo string) (*model.WorldInviteModel, error) {
	if !IsWorldAdmin(worldID, creatorID) {
		return nil, ErrWorldPermission
	}
	db := model.GetDB()
	if err := db.Model(&model.WorldInviteModel{}).
		Where("world_id = ? AND status = ?", worldID, "active").
		Updates(map[string]any{"status": "archived", "updated_at": time.Now()}).Error; err != nil {
		return nil, err
	}
	invite := &model.WorldInviteModel{
		WorldID:   worldID,
		CreatorID: creatorID,
		MaxUse:    maxUse,
		Memo:      memo,
		Status:    "active",
	}
	if ttlMinutes > 0 {
		expire := time.Now().Add(time.Duration(ttlMinutes) * time.Minute)
		invite.ExpireAt = &expire
	}
	if maxUse < 0 {
		invite.MaxUse = 0
	}
	if err := db.Create(invite).Error; err != nil {
		return nil, err
	}
	return invite, nil
}

func WorldInviteConsume(slug, userID string) (*model.WorldInviteModel, *model.WorldModel, *model.WorldMemberModel, bool, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, nil, nil, false, ErrWorldInviteInvalid
	}
	db := model.GetDB()
	var invite model.WorldInviteModel
	if err := db.Where("slug = ? AND status = ?", slug, "active").Limit(1).Find(&invite).Error; err != nil {
		return nil, nil, nil, false, err
	}
	if invite.ID == "" {
		return nil, nil, nil, false, ErrWorldInviteInvalid
	}
	if invite.ExpireAt != nil && invite.ExpireAt.Before(time.Now()) {
		return nil, nil, nil, false, ErrWorldInviteInvalid
	}
	if invite.MaxUse > 0 && invite.UsedCount >= invite.MaxUse {
		return nil, nil, nil, false, ErrWorldInviteInvalid
	}
	world, err := GetWorldByID(invite.WorldID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	existingMember := &model.WorldMemberModel{}
	_ = db.Where("world_id = ? AND user_id = ?", invite.WorldID, userID).Limit(1).Find(existingMember).Error
	wasMember := existingMember.ID != ""
	member, err := WorldJoin(invite.WorldID, userID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	alreadyJoined := wasMember
	if !wasMember {
		_ = db.Model(&model.WorldInviteModel{}).
			Where("id = ?", invite.ID).
			Updates(map[string]any{"used_count": gorm.Expr("used_count + 1"), "updated_at": time.Now()}).Error
	}
	return &invite, world, member, alreadyJoined, nil
}
