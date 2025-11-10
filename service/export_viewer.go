package service

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sealchat/model"
)

type viewerManifest struct {
	ChannelID      string                 `json:"channel_id"`
	ChannelName    string                 `json:"channel_name"`
	GeneratedAt    time.Time              `json:"generated_at"`
	DisplayOptions map[string]any         `json:"display_options,omitempty"`
	SliceLimit     int                    `json:"slice_limit"`
	MaxConcurrency int                    `json:"max_concurrency"`
	PartTotal      int                    `json:"part_total"`
	TotalMessages  int                    `json:"total_messages"`
	Parts          []viewerManifestPart   `json:"parts"`
	Meta           map[string]interface{} `json:"meta,omitempty"`
}

type viewerManifestPart struct {
	File       string     `json:"file"`
	PartIndex  int        `json:"part_index"`
	PartTotal  int        `json:"part_total"`
	Messages   int        `json:"messages"`
	SliceStart *time.Time `json:"slice_start,omitempty"`
	SliceEnd   *time.Time `json:"slice_end,omitempty"`
	SHA256     string     `json:"sha256,omitempty"`
}

type partRenderResult struct {
	fileName string
	content  []byte
	meta     viewerManifestPart
	err      error
}

func processViewerExportJob(
	job *model.MessageExportJobModel,
	channelName string,
	messages []*model.MessageModel,
	storageDir string,
	extra *exportExtraOptions,
) error {
	if extra == nil {
		extra = parseExportExtraOptions("")
	}
	chunks := splitMessagesForViewer(messages, extra.SliceLimit)
	partTotal := len(chunks)
	generatedAt := time.Now()
	assets := getViewerAssets()
	embedder := newInlineImageEmbedder()
	results := make([]partRenderResult, partTotal)

	if err := renderViewerParts(job, channelName, chunks, assets, extra, generatedAt, results, embedder); err != nil {
		return err
	}

	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("创建导出目录失败: %w", err)
	}

	fileName := buildViewerArchiveName(channelName, generatedAt)
	filePath := filepath.Join(storageDir, fileName)

	if err := writeViewerArchive(filePath, job, channelName, extra, generatedAt, results, assets); err != nil {
		return err
	}

	return markJobDone(job, filePath, fileName, "zip")
}

func splitMessagesForViewer(messages []*model.MessageModel, limit int) [][]*model.MessageModel {
	if limit <= 0 {
		limit = DefaultExportSliceLimit
	}
	if limit < MinExportSliceLimit {
		limit = MinExportSliceLimit
	}
	if limit > MaxExportSliceLimit {
		limit = MaxExportSliceLimit
	}
	if len(messages) == 0 {
		return [][]*model.MessageModel{make([]*model.MessageModel, 0)}
	}
	var chunks [][]*model.MessageModel
	for i := 0; i < len(messages); i += limit {
		end := i + limit
		if end > len(messages) {
			end = len(messages)
		}
		chunk := messages[i:end]
		chunks = append(chunks, chunk)
	}
	if len(chunks) == 0 {
		chunks = append(chunks, make([]*model.MessageModel, 0))
	}
	return chunks
}

func renderViewerParts(
	job *model.MessageExportJobModel,
	channelName string,
	chunks [][]*model.MessageModel,
	assets viewerAssets,
	extra *exportExtraOptions,
	generatedAt time.Time,
	results []partRenderResult,
	embedder *inlineImageEmbedder,
) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 1)
	sem := make(chan struct{}, normalizeConcurrency(extra.MaxConcurrency))

	for idx, chunk := range chunks {
		wg.Add(1)
		go func(index int, messages []*model.MessageModel) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			partFile := fmt.Sprintf("parts/part-%03d.html", index+1)
			start, end := sliceBounds(messages)
			ctx := &payloadContext{
				DisplayOptions: extra.DisplaySettings,
				PartIndex:      index + 1,
				PartTotal:      len(chunks),
				SliceStart:     start,
				SliceEnd:       end,
				GeneratedAt:    &generatedAt,
			}
			payload := buildExportPayload(job, channelName, messages, ctx)
			if embedder != nil {
				embedder.inlinePayload(payload)
			}
			htmlBytes, err := renderHTMLPart(payload, assets)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			hash := sha256.Sum256(htmlBytes)
			results[index] = partRenderResult{
				fileName: partFile,
				content:  htmlBytes,
				meta: viewerManifestPart{
					File:       partFile,
					PartIndex:  index + 1,
					PartTotal:  len(chunks),
					Messages:   len(messages),
					SliceStart: start,
					SliceEnd:   end,
					SHA256:     hex.EncodeToString(hash[:]),
				},
			}
		}(idx, chunk)
	}

	wg.Wait()
	close(errCh)
	if err, ok := <-errCh; ok && err != nil {
		return err
	}
	return nil
}

func writeViewerArchive(
	filePath string,
	job *model.MessageExportJobModel,
	channelName string,
	extra *exportExtraOptions,
	generatedAt time.Time,
	results []partRenderResult,
	assets viewerAssets,
) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建 ZIP 文件失败: %w", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	totalMessages := 0
	for _, result := range results {
		if err := writeZipEntry(zw, result.fileName, result.content); err != nil {
			return err
		}
		totalMessages += result.meta.Messages
	}

	manifest := &viewerManifest{
		ChannelID:      job.ChannelID,
		ChannelName:    channelName,
		GeneratedAt:    generatedAt.UTC(),
		DisplayOptions: cloneDisplayOptions(&payloadContext{DisplayOptions: extra.DisplaySettings}),
		SliceLimit:     extra.SliceLimit,
		MaxConcurrency: extra.MaxConcurrency,
		PartTotal:      len(results),
		TotalMessages:  totalMessages,
		Parts:          make([]viewerManifestPart, len(results)),
	}
	for idx, result := range results {
		manifest.Parts[idx] = result.meta
	}

	metaBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 manifest 失败: %w", err)
	}
	if err := writeZipEntry(zw, "manifest/meta.json", metaBytes); err != nil {
		return err
	}

	indexBytes, err := renderViewerIndex(manifest, assets)
	if err != nil {
		return err
	}
	if err := writeZipEntry(zw, "index.html", indexBytes); err != nil {
		return err
	}

	return nil
}

func writeZipEntry(zw *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{
		Name:   name,
		Method: zip.Deflate,
	}
	writer, err := zw.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("写入 ZIP 失败 (%s): %w", name, err)
	}
	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("写入 ZIP 内容失败 (%s): %w", name, err)
	}
	return nil
}

func sliceBounds(messages []*model.MessageModel) (*time.Time, *time.Time) {
	if len(messages) == 0 {
		return nil, nil
	}
	start := messages[0].CreatedAt
	end := messages[len(messages)-1].CreatedAt
	return &start, &end
}

func normalizeConcurrency(value int) int {
	value = NormalizeExportConcurrency(value)
	if value <= 0 {
		value = 1
	}
	return value
}

func buildViewerArchiveName(channelName string, generatedAt time.Time) string {
	safeName := sanitizeFileName(channelName)
	if safeName == "" {
		safeName = "channel"
	}
	return fmt.Sprintf("sealchat-%s-%s.zip", safeName, generatedAt.Format("20060102-1504"))
}
