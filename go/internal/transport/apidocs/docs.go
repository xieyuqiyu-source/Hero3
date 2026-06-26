// Package apidocs 提供随 Go 服务启动的在线接口文档页面。
package apidocs

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	docsPath        = "/docs"
	docsSlashPath   = "/docs/"
	openAPIYAMLPath = "/openapi.yaml"
	defaultSpecPath = "docs/接口文档/openapi打包.yaml"
	openAPIPathEnv  = "HERO3_OPENAPI_PATH"
	scalarScriptURL = "https://cdn.jsdelivr.net/npm/@scalar/api-reference"
	specContentType = "application/yaml; charset=utf-8"
	htmlContentType = "text/html; charset=utf-8"
)

// Options 定义在线接口文档的注册参数。
type Options struct {
	SpecPath string
}

// Handler 负责返回接口文档页面和 OpenAPI 文件。
type Handler struct {
	specPath string
}

// RegisterRoutes 把在线接口文档路由注册到 HTTP mux。
func RegisterRoutes(mux *http.ServeMux, options Options) {
	handler := NewHandler(options)
	mux.HandleFunc("GET "+docsPath, handler.Docs)
	mux.HandleFunc("GET "+docsSlashPath, handler.Docs)
	mux.HandleFunc("GET "+openAPIYAMLPath, handler.OpenAPIYAML)
}

// PublicPaths 返回在线接口文档需要跳过登录认证的公开路径。
func PublicPaths() []string {
	return []string{docsPath, docsSlashPath, openAPIYAMLPath}
}

// NewHandler 创建在线接口文档处理器。
func NewHandler(options Options) *Handler {
	return &Handler{specPath: strings.TrimSpace(options.SpecPath)}
}

// Docs 返回 Scalar 接口文档调试页面。
func (h *Handler) Docs(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == docsSlashPath {
		http.Redirect(w, r, docsPath, http.StatusMovedPermanently)
		return
	}

	w.Header().Set("Content-Type", htmlContentType)
	_, _ = fmt.Fprint(w, docsHTML())
}

// OpenAPIYAML 返回当前打包后的 OpenAPI YAML 文档。
func (h *Handler) OpenAPIYAML(w http.ResponseWriter, r *http.Request) {
	specPath, err := h.resolveSpecPath()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	content, err := os.ReadFile(specPath)
	if err != nil {
		http.Error(w, "读取 OpenAPI 文档失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", specContentType)
	_, _ = w.Write(content)
}

// resolveSpecPath 根据显式配置、环境变量和常见运行目录查找 OpenAPI 文件。
func (h *Handler) resolveSpecPath() (string, error) {
	candidates := make([]string, 0, 4)
	if h.specPath != "" {
		candidates = append(candidates, h.specPath)
	}
	if envPath := strings.TrimSpace(os.Getenv(openAPIPathEnv)); envPath != "" {
		candidates = append(candidates, envPath)
	}
	candidates = append(candidates, defaultSpecPath, filepath.Join("..", defaultSpecPath))

	for _, candidate := range candidates {
		if isReadableFile(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("未找到 OpenAPI 文档，请先运行 make openapi 或设置 %s", openAPIPathEnv)
}

// isReadableFile 判断目标路径是否是可读取文件。
func isReadableFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// docsHTML 生成 Scalar 在线接口文档页面。
func docsHTML() string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Hero3 API Docs</title>
  <style>
    body {
      margin: 0;
      background: #f7f8fb;
    }
  </style>
</head>
<body>
  <div id="app"></div>
  <script src="` + scalarScriptURL + `"></script>
  <script>
    Scalar.createApiReference('#app', {
      url: '` + openAPIYAMLPath + `',
    })
  </script>
</body>
</html>`
}
