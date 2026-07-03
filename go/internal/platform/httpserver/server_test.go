// 本文件测试 HTTP 服务中间件的请求日志和异常恢复行为。
package httpserver

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequestLogMiddlewareRecordsStatusAndBytes 验证请求日志会记录状态码和响应字节数。
func TestRequestLogMiddlewareRecordsStatusAndBytes(t *testing.T) {
	logger, logs := newTestLogger()
	handler := requestLogMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	output := logs.String()
	for _, want := range []string{"request handled", "method=POST", "path=/api/v1/test", "status=201", "bytes=2", "duration_ms="} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected log to contain %q, got %s", want, output)
		}
	}
}

// TestRequestLogMiddlewareDefaultsStatusOK 验证 handler 不写响应时日志按 200 记录。
func TestRequestLogMiddlewareDefaultsStatusOK(t *testing.T) {
	logger, logs := newTestLogger()
	handler := requestLogMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if output := logs.String(); !strings.Contains(output, "status=200") {
		t.Fatalf("expected log to contain status=200, got %s", output)
	}
}

// TestRequestLogMiddlewareRecordsRecoveredPanic 验证 panic 被恢复后请求日志仍记录 500。
func TestRequestLogMiddlewareRecordsRecoveredPanic(t *testing.T) {
	logger, logs := newTestLogger()
	handler := requestLogMiddleware(logger, recoverMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	output := logs.String()
	for _, want := range []string{"panic recovered", "request handled", "path=/api/v1/panic", "status=500"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected log to contain %q, got %s", want, output)
		}
	}
}

// newTestLogger 创建写入内存缓冲区的测试日志器。
func newTestLogger() (*slog.Logger, *bytes.Buffer) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return logger, &logs
}
