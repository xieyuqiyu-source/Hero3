// Package helpdocs 负责读取项目内可手动维护的帮助文档。
package helpdocs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultContentDir = "helpdocs/content"

// DocumentSummary 是帮助文档列表里的简要信息。
type DocumentSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Excerpt   string    `json:"excerpt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Document 是单篇帮助文档的完整内容。
type Document struct {
	DocumentSummary
	Content string `json:"content"`
}

// Service 提供帮助文档读取和索引能力。
type Service struct {
	contentDir string
}

// NewService 创建帮助文档服务。
func NewService(contentDir string) *Service {
	return &Service{contentDir: strings.TrimSpace(contentDir)}
}

// ListDocuments 扫描帮助文档目录并返回文档摘要。
func (s *Service) ListDocuments() ([]DocumentSummary, error) {
	contentDir, err := s.resolveContentDir()
	if err != nil {
		return nil, err
	}

	summaries := make([]DocumentSummary, 0)
	if err := filepath.WalkDir(contentDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}

		document, err := s.readDocumentAt(contentDir, path)
		if err != nil {
			return err
		}
		summaries = append(summaries, document.DocumentSummary)
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ID < summaries[j].ID
	})
	return summaries, nil
}

// GetDocument 按文档 ID 读取单篇帮助文档。
func (s *Service) GetDocument(id string) (Document, error) {
	contentDir, err := s.resolveContentDir()
	if err != nil {
		return Document{}, err
	}

	cleanID, err := cleanDocumentID(id)
	if err != nil {
		return Document{}, err
	}

	path := filepath.Join(contentDir, filepath.FromSlash(cleanID)+".md")
	document, err := s.readDocumentAt(contentDir, path)
	if err != nil {
		return Document{}, err
	}
	return document, nil
}

// resolveContentDir 根据配置和常见运行目录定位帮助文档目录。
func (s *Service) resolveContentDir() (string, error) {
	candidates := make([]string, 0, 3)
	if s.contentDir != "" {
		candidates = append(candidates, s.contentDir)
	}
	candidates = append(candidates, defaultContentDir, filepath.Join("..", defaultContentDir))

	for _, candidate := range candidates {
		if isReadableDir(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("未找到帮助文档目录：%s", defaultContentDir)
}

// readDocumentAt 读取并解析指定 Markdown 文件。
func (s *Service) readDocumentAt(contentDir string, path string) (Document, error) {
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(contentDir)) {
		return Document{}, errors.New("帮助文档路径非法")
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Document{}, err
	}

	id, err := documentIDFromPath(contentDir, path)
	if err != nil {
		return Document{}, err
	}

	content := string(contentBytes)
	title := extractTitle(content, id)
	return Document{
		DocumentSummary: DocumentSummary{
			ID:        id,
			Title:     title,
			Excerpt:   extractExcerpt(content),
			UpdatedAt: info.ModTime(),
		},
		Content: content,
	}, nil
}

// cleanDocumentID 校验并清理文档 ID，避免目录穿越。
func cleanDocumentID(id string) (string, error) {
	cleaned := strings.Trim(strings.TrimSpace(id), "/")
	if cleaned == "" || strings.Contains(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", errors.New("帮助文档 ID 非法")
	}
	return filepath.ToSlash(filepath.Clean(cleaned)), nil
}

// documentIDFromPath 把文件路径转换为前端使用的文档 ID。
func documentIDFromPath(contentDir string, path string) (string, error) {
	relative, err := filepath.Rel(contentDir, path)
	if err != nil {
		return "", err
	}
	withoutExt := strings.TrimSuffix(relative, filepath.Ext(relative))
	return filepath.ToSlash(withoutExt), nil
}

// extractTitle 从 Markdown 一级标题提取文档标题。
func extractTitle(content string, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return fallback
}

// extractExcerpt 提取第一段普通文本作为摘要。
func extractExcerpt(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		if len([]rune(trimmed)) > 90 {
			return string([]rune(trimmed)[:90]) + "..."
		}
		return trimmed
	}
	return "暂无摘要"
}

// isReadableDir 判断路径是否是可读目录。
func isReadableDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
