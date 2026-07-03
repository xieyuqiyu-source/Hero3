// 本文件验证 API 通用响应工具的边界行为。
package api

import (
	"errors"
	"net/http"
	"testing"
)

type failingResponseWriter struct {
	header http.Header
	status []int
}

// Header 返回测试用响应头集合。
func (w *failingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

// Write 模拟响应体写入失败。
func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

// WriteHeader 记录每一次状态码写入。
func (w *failingResponseWriter) WriteHeader(statusCode int) {
	w.status = append(w.status, statusCode)
}

// TestWriteJSONDoesNotRewriteHeaderAfterBodyWriteFailure 验证响应头发出后写失败不会二次写 500。
func TestWriteJSONDoesNotRewriteHeaderAfterBodyWriteFailure(t *testing.T) {
	writer := &failingResponseWriter{}

	writeJSON(writer, http.StatusOK, map[string]string{"ok": "true"})

	if len(writer.status) != 1 {
		t.Fatalf("expected one WriteHeader call, got %d: %v", len(writer.status), writer.status)
	}
	if writer.status[0] != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, writer.status[0])
	}
}
