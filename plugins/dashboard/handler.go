package dashboard

import (
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/xraph/forge"
)

// Handler handles dashboard SPA requests
type Handler struct {
	assets fs.FS
}

// NewHandler creates a new dashboard handler
func NewHandler(assets fs.FS) *Handler {
	h := &Handler{
		assets: assets,
	}
	// Debug: log available files in embedded FS
	h.debugAssets()
	return h
}

// debugAssets prints all available files in the embedded filesystem
func (h *Handler) debugAssets() {
	if h.assets == nil {
		log.Println("[Dashboard] Warning: assets filesystem is nil")
		return
	}

	log.Println("[Dashboard] Available files in embedded filesystem:")
	fs.WalkDir(h.assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			log.Printf("[Dashboard]   %s", path)
		}
		return nil
	})
}

// ServeAssets serves static assets for the dashboard SPA
func (h *Handler) ServeAssets(c *forge.Context) error {
	// Get the requested path, removing the /dashboard prefix
	path := strings.TrimPrefix(c.Request().URL.Path, "/dashboard/")

	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	log.Printf("[Dashboard] ServeAssets: requested path '%s'", path)

	// Security: prevent directory traversal
	if strings.Contains(path, "..") {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid path"})
	}

	// If path is empty or just "/", serve index
	if path == "" {
		return h.ServeIndex(c)
	}

	// Try to read the file
	content, err := fs.ReadFile(h.assets, path)
	if err != nil {
		log.Printf("[Dashboard] File not found: %s, error: %v", path, err)
		// If file not found, serve index.html for SPA routing
		return h.ServeIndex(c)
	}

	log.Printf("[Dashboard] Serving file: %s (%d bytes)", path, len(content))

	// Set appropriate content type based on file extension
	ext := filepath.Ext(path)
	contentType := getContentType(ext)
	c.Header().Set("Content-Type", contentType)
	c.Header().Set("Cache-Control", "public, max-age=31536000")

	// Write the content
	return c.HTML(http.StatusOK, string(content))
}

// ServeDashboardAssets serves static assets from the /dashboard/assets/ path
func (h *Handler) ServeDashboardAssets(c *forge.Context) error {
	// Get the file path from the URL, removing the /dashboard/assets/ prefix
	urlPath := c.Request().URL.Path
	log.Printf("[Dashboard] ServeDashboardAssets: full URL path '%s'", urlPath)

	filePath := strings.TrimPrefix(urlPath, "/dashboard/assets/")
	if filePath == "" || filePath == urlPath {
		log.Printf("[Dashboard] Failed to remove prefix from path: %s", urlPath)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "asset not found"})
	}

	// Try both with and without the assets/ prefix
	paths := []string{
		"assets/" + filePath,
		filePath,
	}

	var content []byte
	var err error
	var successPath string

	for _, assetPath := range paths {
		log.Printf("[Dashboard] Trying to read: %s", assetPath)
		content, err = fs.ReadFile(h.assets, assetPath)
		if err == nil {
			successPath = assetPath
			break
		}
		log.Printf("[Dashboard]   Not found: %v", err)
	}

	if err != nil {
		log.Printf("[Dashboard] Asset not found after trying all paths: %s", filePath)
		return c.JSON(http.StatusNotFound, map[string]string{"error": "asset not found"})
	}

	log.Printf("[Dashboard] Serving asset: %s (%d bytes)", successPath, len(content))

	// Set appropriate content type based on file extension
	ext := filepath.Ext(filePath)
	contentType := getContentType(ext)
	c.Header().Set("Content-Type", contentType)
	c.Header().Set("Cache-Control", "public, max-age=31536000")

	return c.HTML(http.StatusOK, string(content))
}

// ServeIndex serves the index.html file for SPA routing
func (h *Handler) ServeIndex(c *forge.Context) error {
	log.Println("[Dashboard] ServeIndex called")

	// Check if assets filesystem is available
	if h.assets == nil {
		log.Println("[Dashboard] Assets filesystem is nil, serving fallback")
		return c.HTML(http.StatusOK, getFallbackHTML())
	}

	// Read index.html from embedded assets
	content, err := fs.ReadFile(h.assets, "index.html")
	if err != nil {
		log.Printf("[Dashboard] Failed to read index.html: %v, serving fallback", err)
		return c.HTML(http.StatusOK, getFallbackHTML())
	}

	log.Printf("[Dashboard] Serving index.html (%d bytes)", len(content))

	// Update asset paths to be scoped under dashboard
	htmlContent := string(content)
	htmlContent = strings.ReplaceAll(htmlContent, `"/assets/`, `"/dashboard/assets/`)
	htmlContent = strings.ReplaceAll(htmlContent, `'/assets/`, `'/dashboard/assets/`)
	htmlContent = strings.ReplaceAll(htmlContent, `href="/`, `href="/dashboard/`)
	htmlContent = strings.ReplaceAll(htmlContent, `src="/`, `src="/dashboard/`)

	// Fix double replacements
	htmlContent = strings.ReplaceAll(htmlContent, `"/dashboard/dashboard/`, `"/dashboard/`)

	c.Header().Set("Content-Type", "text/html; charset=utf-8")
	return c.HTML(http.StatusOK, htmlContent)
}

// getFallbackHTML returns a basic HTML page when assets are not available
func getFallbackHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AuthSome Dashboard</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            background: white;
            border-radius: 12px;
            padding: 3rem;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
            max-width: 500px;
            margin: 2rem;
        }
        .logo { font-size: 3rem; margin-bottom: 1rem; }
        h1 { color: #333; margin-bottom: 1rem; font-size: 2.5rem; font-weight: 700; }
        p { color: #666; font-size: 1.1rem; line-height: 1.6; margin-bottom: 2rem; }
        .status {
            background: #e8f5e8;
            color: #2d5a2d;
            padding: 1rem;
            border-radius: 8px;
            margin-bottom: 2rem;
            border-left: 4px solid #4caf50;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">🔐</div>
        <h1>AuthSome Dashboard</h1>
        <div class="status">✅ Dashboard plugin is running successfully!</div>
        <p>Welcome to the AuthSome authentication framework dashboard. This is a fallback page that confirms the dashboard plugin is properly loaded and functioning.</p>
        <p>The full dashboard interface will be available once the frontend assets are built.</p>
    </div>
</body>
</html>`
}

// getContentType returns the appropriate content type for file extensions
func getContentType(ext string) string {
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	case ".webp":
		return "image/webp"
	case ".wasm":
		return "application/wasm"
	default:
		return "application/octet-stream"
	}
}

// HTTP-compatible handlers for pure http.ServeMux routing

// ServeDashboardAssetsHTTP serves dashboard assets for pure http.ServeMux
func (h *Handler) ServeDashboardAssetsHTTP(w http.ResponseWriter, r *http.Request) {
	// Remove /dashboard/assets/ prefix to get the asset path
	assetPath := strings.TrimPrefix(r.URL.Path, "/dashboard/assets/")
	if assetPath == "" {
		http.NotFound(w, r)
		return
	}

	// Prevent directory traversal
	if strings.Contains(assetPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Try to read the file from assets/
	fullPath := "assets/" + assetPath
	content, err := fs.ReadFile(h.assets, fullPath)
	if err != nil {
		log.Printf("[Dashboard] Asset not found: %s (full path: %s)", assetPath, fullPath)
		http.NotFound(w, r)
		return
	}

	// Set content type based on file extension
	ext := filepath.Ext(assetPath)
	contentType := getContentType(ext)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000") // 1 year cache for assets

	w.Write(content)
}

// ServeIndexHTTP serves the dashboard index page for pure http.ServeMux
func (h *Handler) ServeIndexHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to read index.html
	content, err := fs.ReadFile(h.assets, "index.html")
	if err != nil {
		log.Printf("[Dashboard] index.html not found: %v", err)
		// Serve fallback HTML
		content = []byte(getFallbackHTML())
	}

	// Modify the HTML content to use /dashboard/assets/ prefix
	htmlContent := string(content)
	htmlContent = strings.ReplaceAll(htmlContent, `"/assets/`, `"/dashboard/assets/`)
	htmlContent = strings.ReplaceAll(htmlContent, `'/assets/`, `'/dashboard/assets/`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent))
}

// ServeAssetsHTTP serves other dashboard assets for pure http.ServeMux
func (h *Handler) ServeAssetsHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the requested path, removing the /dashboard prefix
	path := strings.TrimPrefix(r.URL.Path, "/dashboard/")

	// Prevent directory traversal
	if strings.Contains(path, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Try to read the file
	content, err := fs.ReadFile(h.assets, path)
	if err != nil {
		// If file not found, serve index.html (SPA fallback)
		h.ServeIndexHTTP(w, r)
		return
	}

	// Set content type based on file extension
	ext := filepath.Ext(path)
	contentType := getContentType(ext)
	w.Header().Set("Content-Type", contentType)

	w.Write(content)
}
