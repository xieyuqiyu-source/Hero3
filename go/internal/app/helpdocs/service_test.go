// Package helpdocs 测试帮助文档服务的目录扫描和文档读取能力。
package helpdocs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestServiceReadsNestedDocuments 验证嵌套目录里的帮助文档可以被列出和读取。
func TestServiceReadsNestedDocuments(t *testing.T) {
	contentDir := t.TempDir()
	nestedDir := filepath.Join(contentDir, "01-project")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "overview.md"), []byte("# 项目总览\n\n这里是摘要。"), 0o644); err != nil {
		t.Fatalf("write document: %v", err)
	}

	service := NewService(contentDir)
	documents, err := service.ListDocuments()
	if err != nil {
		t.Fatalf("list documents: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(documents))
	}
	if documents[0].ID != "01-project/overview" {
		t.Fatalf("unexpected document id: %s", documents[0].ID)
	}

	document, err := service.GetDocument("01-project/overview")
	if err != nil {
		t.Fatalf("get document: %v", err)
	}
	if document.Title != "项目总览" {
		t.Fatalf("unexpected title: %s", document.Title)
	}
}

// TestServiceRejectsInvalidDocumentID 验证帮助文档 ID 不能目录穿越。
func TestServiceRejectsInvalidDocumentID(t *testing.T) {
	service := NewService(t.TempDir())
	if _, err := service.GetDocument("../secret"); err == nil {
		t.Fatal("expected invalid document id error")
	}
}
