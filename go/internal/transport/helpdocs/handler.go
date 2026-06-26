// Package helpdocs 提供帮助文档站的 HTTP API。
package helpdocs

import (
	"encoding/json"
	"net/http"
	"strings"

	apphelpdocs "hero3/internal/app/helpdocs"
)

const docsAPIPath = "/api/v1/help/docs"

// Handler 负责处理帮助文档 HTTP 请求。
type Handler struct {
	service *apphelpdocs.Service
}

// RegisterRoutes 注册帮助文档 API 路由。
func RegisterRoutes(mux *http.ServeMux, service *apphelpdocs.Service) {
	handler := &Handler{service: service}
	mux.HandleFunc("GET "+docsAPIPath, handler.ListDocuments)
	mux.HandleFunc("GET "+docsAPIPath+"/{docId...}", handler.GetDocument)
}

// ListDocuments 返回帮助文档摘要列表。
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	documents, err := h.service.ListDocuments()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": documents})
}

// GetDocument 返回单篇帮助文档内容。
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	documentID := strings.TrimSpace(r.PathValue("docId"))
	document, err := h.service.GetDocument(documentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": document})
}

// writeJSON 统一写出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
