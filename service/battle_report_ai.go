package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"sealchat/model"
	aiService "sealchat/service/ai"
	"sealchat/utils"
)

type BattleReportSummaryInput struct {
	Title              string
	PeriodStart        time.Time
	PeriodEnd          time.Time
	ContextReportCount int
	SourceChannelIDs   []string
	Source             string
	AIConfig           utils.AIConfig
	Runner             aiService.TaskRunner
}

type BattleReportSummaryRunOptions struct {
	User             *model.UserModel
	Source           string
	SourceChannelIDs []string
	AIConfig         utils.AIConfig
	Runner           aiService.TaskRunner
}

func StartBattleReportSummary(ctx context.Context, channelID string, userID string, input BattleReportSummaryInput) (*model.BattleReportModel, error) {
	channels, err := resolveBattleReportSourceChannels(channelID, "", input.SourceChannelIDs, userID)
	if err != nil {
		return nil, err
	}
	sourceChannelIDs := make([]string, 0, len(channels))
	for _, channel := range channels {
		sourceChannelIDs = append(sourceChannelIDs, channel.ID)
	}
	item, err := CreateBattleReport(channelID, userID, BattleReportInput{
		Title:              input.Title,
		PeriodStart:        input.PeriodStart,
		PeriodEnd:          input.PeriodEnd,
		ContextReportCount: input.ContextReportCount,
		Status:             model.BattleReportStatusGenerating,
		AISource:           input.Source,
		AIFeatureKey:       aiService.FeatureBattleSummary,
	})
	if err != nil {
		return nil, err
	}
	user := model.UserGet(userID)
	if user == nil {
		user = &model.UserModel{StringPKBaseModel: model.StringPKBaseModel{ID: userID}}
	}
	go func() {
		if err := runBattleReportSummaryTask(ctx, item.ID, BattleReportSummaryRunOptions{
			User:             user,
			Source:           input.Source,
			SourceChannelIDs: sourceChannelIDs,
			AIConfig:         input.AIConfig,
			Runner:           input.Runner,
		}); err != nil {
			_ = markBattleReportSummaryFailed(item.ID, err)
		}
	}()
	return item, nil
}

func runBattleReportSummaryTask(ctx context.Context, reportID string, opts BattleReportSummaryRunOptions) error {
	report, err := loadBattleReport(reportID)
	if err != nil {
		return err
	}
	if opts.User == nil {
		return fmt.Errorf("缺少用户信息")
	}
	channels, err := resolveBattleReportSourceChannels(report.ChannelID, report.WorldID, opts.SourceChannelIDs, opts.User.ID)
	if err != nil {
		return markBattleReportSummaryFailed(report.ID, err)
	}
	messageGroups, err := loadBattleReportMessageGroups(channels, report.PeriodStart, report.PeriodEnd)
	if err != nil {
		return err
	}
	if battleReportMessageGroupLen(messageGroups) == 0 {
		return markBattleReportSummaryFailed(report.ID, fmt.Errorf("所选时间范围内没有可总结的消息"))
	}
	contextReports, err := loadBattleReportContextReports(report)
	if err != nil {
		return err
	}
	contextReports, messageGroups = limitBattleReportSummaryPromptInput(
		report,
		contextReports,
		messageGroups,
		battleReportSummaryMaxInputTokens(opts.AIConfig),
	)
	if battleReportMessageGroupLen(messageGroups) == 0 {
		return markBattleReportSummaryFailed(report.ID, fmt.Errorf("战报总结最大输入 token 数过低，没有可总结的消息"))
	}
	prompt := buildBattleReportSummaryPromptWithGroups(report, contextReports, messageGroups)
	output, err := aiService.RunTaskWithBilling(ctx, aiService.BilledRunInput{
		Config:     opts.AIConfig,
		User:       opts.User,
		FeatureKey: aiService.FeatureBattleSummary,
		WorldID:    report.WorldID,
		Input:      prompt,
		Source:     opts.Source,
		Runner:     opts.Runner,
	})
	if err != nil {
		return markBattleReportSummaryFailed(report.ID, err)
	}
	result := strings.TrimSpace(output.Result.Result)
	if result == "" {
		return markBattleReportSummaryFailed(report.ID, fmt.Errorf("AI 返回空战报"))
	}
	updates := map[string]interface{}{
		"content":         result,
		"content_preview": model.BuildBattleReportPreview(result, 200),
		"status":          model.BattleReportStatusReady,
		"error_message":   "",
		"ai_source":       strings.TrimSpace(opts.Source),
		"ai_provider_id":  output.Result.ProviderID,
		"ai_model":        output.Result.Model,
		"ai_feature_key":  aiService.FeatureBattleSummary,
	}
	return model.GetDB().Model(&model.BattleReportModel{}).
		Where("id = ? AND is_deleted = ?", report.ID, false).
		Updates(updates).Error
}

func resolveBattleReportSourceChannels(primaryChannelID string, worldID string, sourceChannelIDs []string, userID string) ([]*model.ChannelModel, error) {
	primary, err := loadBattleReportChannel(primaryChannelID)
	if err != nil {
		return nil, err
	}
	targetWorldID := strings.TrimSpace(worldID)
	if targetWorldID == "" {
		targetWorldID = primary.WorldID
	}
	ids := make([]string, 0, len(sourceChannelIDs)+1)
	seen := map[string]struct{}{}
	addID := func(raw string) {
		id := strings.TrimSpace(raw)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for _, id := range sourceChannelIDs {
		addID(id)
	}
	if len(ids) == 0 {
		addID(primary.ID)
	}

	channels := make([]*model.ChannelModel, 0, len(ids))
	for _, id := range ids {
		channel, err := loadBattleReportChannel(id)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(channel.WorldID) != targetWorldID {
			return nil, fmt.Errorf("战报来源频道必须属于同一世界")
		}
		if userID != "" && !CanReadChannelByUserId(userID, channel.ID) {
			return nil, fmt.Errorf("无权读取战报来源频道")
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

func loadBattleReportMessageGroups(channels []*model.ChannelModel, start time.Time, end time.Time) ([]BattleReportMessageGroup, error) {
	groups := make([]BattleReportMessageGroup, 0, len(channels))
	for _, channel := range channels {
		if channel == nil {
			continue
		}
		messages, err := loadBattleReportMessages(channel.ID, start, end)
		if err != nil {
			return nil, err
		}
		if len(messages) == 0 {
			continue
		}
		groups = append(groups, BattleReportMessageGroup{
			ChannelID:   channel.ID,
			ChannelName: channel.Name,
			Messages:    messages,
		})
	}
	return groups, nil
}

func battleReportMessageGroupLen(groups []BattleReportMessageGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.Messages)
	}
	return total
}

func battleReportSummaryMaxInputTokens(cfg utils.AIConfig) int {
	normalized := utils.NormalizeAIConfig(cfg)
	return normalized.Features[aiService.FeatureBattleSummary].Params.MaxInputTokens
}

func limitBattleReportSummaryPromptInput(report *model.BattleReportModel, contextReports []*model.BattleReportModel, groups []BattleReportMessageGroup, maxInputTokens int) ([]*model.BattleReportModel, []BattleReportMessageGroup) {
	if maxInputTokens <= 0 {
		return contextReports, groups
	}
	if estimateBattleReportInputTokens(buildBattleReportSummaryPromptWithGroups(report, contextReports, groups)) <= maxInputTokens {
		return contextReports, groups
	}
	type indexedMessage struct {
		message      *model.MessageModel
		groupIndex   int
		messageIndex int
	}
	messages := make([]indexedMessage, 0)
	for groupIndex, group := range groups {
		for messageIndex, msg := range group.Messages {
			if msg == nil {
				continue
			}
			messages = append(messages, indexedMessage{
				message:      msg,
				groupIndex:   groupIndex,
				messageIndex: messageIndex,
			})
		}
	}
	sort.SliceStable(messages, func(i, j int) bool {
		left := messages[i]
		right := messages[j]
		if !left.message.CreatedAt.Equal(right.message.CreatedAt) {
			return left.message.CreatedAt.Before(right.message.CreatedAt)
		}
		if left.groupIndex != right.groupIndex {
			return left.groupIndex < right.groupIndex
		}
		return left.messageIndex < right.messageIndex
	})
	kept := map[*model.MessageModel]struct{}{}
	for _, item := range messages {
		kept[item.message] = struct{}{}
	}
	currentReports := contextReports
	currentGroups := buildLimitedBattleReportMessageGroups(groups, kept)
	for _, item := range messages {
		if estimateBattleReportInputTokens(buildBattleReportSummaryPromptWithGroups(report, currentReports, currentGroups)) <= maxInputTokens {
			return currentReports, currentGroups
		}
		delete(kept, item.message)
		currentGroups = buildLimitedBattleReportMessageGroups(groups, kept)
	}
	for len(currentReports) > 0 {
		if estimateBattleReportInputTokens(buildBattleReportSummaryPromptWithGroups(report, currentReports, currentGroups)) <= maxInputTokens {
			return currentReports, currentGroups
		}
		currentReports = currentReports[:len(currentReports)-1]
	}
	return currentReports, currentGroups
}

func buildLimitedBattleReportMessageGroups(groups []BattleReportMessageGroup, kept map[*model.MessageModel]struct{}) []BattleReportMessageGroup {
	out := make([]BattleReportMessageGroup, 0, len(groups))
	for _, group := range groups {
		next := group
		next.Messages = make([]*model.MessageModel, 0, len(group.Messages))
		for _, msg := range group.Messages {
			if _, ok := kept[msg]; ok {
				next.Messages = append(next.Messages, msg)
			}
		}
		if len(next.Messages) > 0 {
			out = append(out, next)
		}
	}
	return out
}

func estimateBattleReportInputTokens(input string) int {
	return len([]rune(strings.TrimSpace(input)))
}

func loadBattleReportMessages(channelID string, start time.Time, end time.Time) ([]*model.MessageModel, error) {
	query := model.GetDB().Model(&model.MessageModel{}).
		Where("channel_id = ?", strings.TrimSpace(channelID)).
		Where("is_deleted = ?", false).
		Where("is_revoked = ?", false).
		Preload("Member").
		Preload("User")
	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("created_at <= ?", end)
	}
	query = query.Order("display_order asc").Order("created_at asc")
	var messages []*model.MessageModel
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	return filterMessagesForBattleReport(messages), nil
}

func filterMessagesForBattleReport(messages []*model.MessageModel) []*model.MessageModel {
	filtered := make([]*model.MessageModel, 0, len(messages))
	for _, msg := range messages {
		if classifyExportMessage(msg, false, false, false).Skip {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func loadBattleReportContextReports(report *model.BattleReportModel) ([]*model.BattleReportModel, error) {
	if report == nil || report.ContextReportCount <= 0 {
		return nil, nil
	}
	var items []*model.BattleReportModel
	err := model.GetDB().
		Where("world_id = ? AND is_deleted = ? AND status = ? AND id <> ?", report.WorldID, false, model.BattleReportStatusReady, report.ID).
		Order("sort_order DESC, period_start DESC, created_at DESC").
		Limit(report.ContextReportCount).
		Find(&items).Error
	return items, err
}

func markBattleReportSummaryFailed(reportID string, cause error) error {
	message := "战报总结失败"
	if cause != nil {
		message = cause.Error()
	}
	return model.GetDB().Model(&model.BattleReportModel{}).
		Where("id = ? AND is_deleted = ?", strings.TrimSpace(reportID), false).
		Updates(map[string]interface{}{
			"status":        model.BattleReportStatusFailed,
			"error_message": message,
		}).Error
}

func ResetGeneratingBattleReportsAfterRestart() error {
	return model.GetDB().Model(&model.BattleReportModel{}).
		Where("status = ? AND is_deleted = ?", model.BattleReportStatusGenerating, false).
		Updates(map[string]interface{}{
			"status":        model.BattleReportStatusFailed,
			"error_message": "服务重启，任务未完成，请重试",
		}).Error
}
