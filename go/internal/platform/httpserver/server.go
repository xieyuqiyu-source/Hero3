// Package httpserver 提供 Hero3 HTTP 服务创建、中间件和请求日志能力。
package httpserver

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"time"

	"hero3/internal/platform/config"
)

const (
	slowRequestWarnThreshold  = 500 * time.Millisecond
	slowRequestErrorThreshold = 2 * time.Second
)

// New 创建 HTTP Server，并按顺序安装恢复和请求日志中间件。
func New(cfg config.Config, logger *slog.Logger, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.Addr,
		Handler:      requestLogMiddleware(logger, recoverMiddleware(logger, handler)),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}

// requestLogMiddleware 记录每个 HTTP 请求的状态码、耗时和响应大小。
func requestLogMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := newResponseLogRecorder(w)
		next.ServeHTTP(recorder, r)

		status := recorder.statusCode()
		duration := time.Since(startedAt)
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"bytes", recorder.bytesWritten,
			"remote", r.RemoteAddr,
		}
		switch {
		case duration >= slowRequestErrorThreshold:
			logger.Error("slow request handled", attrs...)
		case duration >= slowRequestWarnThreshold:
			logger.Warn("slow request handled", attrs...)
		default:
			logger.Info("request handled", attrs...)
		}
	})
}

// recoverMiddleware 捕获 handler panic，避免单个请求导致服务进程退出。
func recoverMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr, "error", recovered)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type responseLogRecorder struct {
	http.ResponseWriter
	status       int
	bytesWritten int
}

// newResponseLogRecorder 创建用于捕获状态码和响应体大小的 ResponseWriter 包装器。
func newResponseLogRecorder(w http.ResponseWriter) *responseLogRecorder {
	return &responseLogRecorder{ResponseWriter: w}
}

// Header 返回底层响应头。
func (r *responseLogRecorder) Header() http.Header {
	return r.ResponseWriter.Header()
}

// WriteHeader 记录最终状态码，并只允许第一次写入生效。
func (r *responseLogRecorder) WriteHeader(statusCode int) {
	if r.status != 0 {
		return
	}
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write 记录响应体字节数，未显式写状态码时按 net/http 语义记为 200。
func (r *responseLogRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytesWritten += n
	return n, err
}

// Flush 转发流式响应刷新能力。
func (r *responseLogRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		if r.status == 0 {
			r.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}

// Hijack 转发连接接管能力。
func (r *responseLogRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

// Unwrap 允许 http.ResponseController 访问底层 ResponseWriter 能力。
func (r *responseLogRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// statusCode 返回最终状态码，handler 未写响应时按 200 记录。
func (r *responseLogRecorder) statusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}
