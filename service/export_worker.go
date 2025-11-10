package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"sealchat/model"
)

type MessageExportWorkerConfig struct {
	StorageDir string
}

var (
	exportWorkerOnce sync.Once
	filenameSafeRe   = regexp.MustCompile(`[^0-9A-Za-z一-龥_-]+`)
)

// StartMessageExportWorker 启动后台导出任务处理协程。
func StartMessageExportWorker(cfg MessageExportWorkerConfig) {
	if cfg.StorageDir == "" {
		cfg.StorageDir = "./data/exports"
	}
	exportWorkerOnce.Do(func() {
		if err := os.MkdirAll(cfg.StorageDir, 0755); err != nil {
			log.Printf("export: 创建导出目录失败: %v", err)
		}
		go runMessageExportWorker(cfg)
	})
}

func runMessageExportWorker(cfg MessageExportWorkerConfig) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		job, err := acquireNextExportJob()
		if err != nil {
			log.Printf("export: 获取任务失败: %v", err)
			<-ticker.C
			continue
		}
		if job == nil {
			<-ticker.C
			continue
		}
		if err := processExportJob(job, cfg.StorageDir); err != nil {
			log.Printf("export: 执行任务 %s 失败: %v", job.ID, err)
		}
	}
}

func acquireNextExportJob() (*model.MessageExportJobModel, error) {
	db := model.GetDB()
	var job model.MessageExportJobModel
	if err := db.Where("status = ?", model.MessageExportStatusPending).
		Order("created_at asc").
		Limit(1).
		Find(&job).Error; err != nil {
		return nil, err
	}
	if job.ID == "" {
		return nil, nil
	}
	res := db.Model(&model.MessageExportJobModel{}).
		Where("id = ? AND status = ?", job.ID, model.MessageExportStatusPending).
		Updates(map[string]any{
			"status":     model.MessageExportStatusProcessing,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, nil
	}
	job.Status = model.MessageExportStatusProcessing
	return &job, nil
}

func processExportJob(job *model.MessageExportJobModel, storageDir string) error {
	channelName := resolveChannelName(job.ChannelID)
	messages, err := loadMessagesForExport(job)
	if err != nil {
		_ = markJobFailed(job, err)
		return err
	}

	extraOptions := parseExportExtraOptions(job.ExtraOptions)
	if strings.EqualFold(job.Format, "html") {
		if err := processViewerExportJob(job, channelName, messages, storageDir, extraOptions); err != nil {
			_ = markJobFailed(job, err)
			return err
		}
		return nil
	}

	payload := buildExportPayload(job, channelName, messages, nil)

	formatter, ok := getFormatter(job.Format)
	if !ok {
		err = fmt.Errorf("不支持的导出格式: %s", job.Format)
		_ = markJobFailed(job, err)
		return err
	}
	data, err := formatter.Build(payload)
	if err != nil {
		_ = markJobFailed(job, err)
		return err
	}

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		_ = markJobFailed(job, err)
		return err
	}

	fileName := buildExportFileName(payload, formatter.Ext())
	filePath := filepath.Join(storageDir, fmt.Sprintf("%s.%s", job.ID, formatter.Ext()))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		_ = markJobFailed(job, err)
		return err
	}

	return markJobDone(job, filePath, fileName)
}

func markJobFailed(job *model.MessageExportJobModel, cause error) error {
	message := ""
	if cause != nil {
		message = cause.Error()
	}
	updates := map[string]any{
		"status":      model.MessageExportStatusFailed,
		"error_msg":   message,
		"finished_at": time.Now(),
	}
	return model.GetDB().Model(&model.MessageExportJobModel{}).
		Where("id = ?", job.ID).
		Updates(updates).Error
}

func markJobDone(job *model.MessageExportJobModel, filePath, fileName string, formatOverride ...string) error {
	updates := map[string]any{
		"status":      model.MessageExportStatusDone,
		"file_path":   filePath,
		"file_name":   fileName,
		"error_msg":   "",
		"finished_at": time.Now(),
	}
	if len(formatOverride) > 0 {
		if format := strings.TrimSpace(formatOverride[0]); format != "" {
			updates["format"] = format
		}
	}
	return model.GetDB().Model(&model.MessageExportJobModel{}).
		Where("id = ?", job.ID).
		Updates(updates).Error
}

func buildExportFileName(payload *ExportPayload, ext string) string {
	base := sanitizeFileName(payload.ChannelName)
	if base == "" {
		base = sanitizeFileName(payload.ChannelID)
	}
	if base == "" {
		base = "channel"
	}
	rangeLabel := safeTimeRangeLabel(payload.StartTime, payload.EndTime)
	if rangeLabel == "" {
		rangeLabel = payload.GeneratedAt.Format("20060102_150405")
	}
	return fmt.Sprintf("%s_%s.%s", base, rangeLabel, ext)
}

func sanitizeFileName(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	return filenameSafeRe.ReplaceAllString(input, "_")
}

func safeTimeRangeLabel(start, end *time.Time) string {
	var parts []string
	if start != nil {
		parts = append(parts, start.Format("20060102"))
	}
	if end != nil {
		parts = append(parts, end.Format("20060102"))
	}
	return strings.Join(parts, "-")
}

func resolveChannelName(channelID string) string {
	if ch, err := model.ChannelGet(channelID); err == nil && ch != nil && strings.TrimSpace(ch.ID) != "" {
		if strings.TrimSpace(ch.Name) != "" {
			return ch.Name
		}
	}
	if fr, err := model.FriendRelationGetByID(channelID); err == nil && fr != nil && strings.TrimSpace(fr.ID) != "" {
		return fmt.Sprintf("私聊-%s-%s", fr.UserID1, fr.UserID2)
	}
	return channelID
}
