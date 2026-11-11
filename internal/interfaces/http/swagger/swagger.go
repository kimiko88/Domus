// Package swagger provides embedded OpenAPI 3.1 specification and Swagger UI interactive documentation.
package swagger

import (
	_ "embed"
	"net/http"
)

//go:embed domus.yaml
var OpenAPISpec []byte

// Handler serves the embedded OpenAPI specification and Swagger UI explorer.
type Handler struct{}

// NewHandler constructs a Swagger handler.
func NewHandler() *Handler {
	return &Handler{}
}

// ServeSpec outputs the raw OpenAPI 3.1 YAML document.
func (h *Handler) ServeSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(OpenAPISpec)
}

// ServeUI serves the embedded Swagger UI documentation page.
func (h *Handler) ServeUI(w http.ResponseWriter, r *http.Request) {
	// Swagger UI assets are currently loaded from unpkg. Scope this exception
	// to the documentation page rather than weakening the global application CSP.
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline' https://unpkg.com; "+
			"style-src 'self' 'unsafe-inline' https://unpkg.com https://fonts.googleapis.com; "+
			"font-src 'self' https://fonts.gstatic.com; "+
			"img-src 'self' data:; "+
			"connect-src 'self' ws: wss:; "+
			"frame-ancestors 'none';",
	)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerHTML))
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Domus REST API — Interactive Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #0b0f19;
      --card: #111827;
      --border: #1f2937;
      --primary: #0ea5e9;
      --text: #f3f4f6;
    }
    html, body {
      margin: 0;
      padding: 0;
      background-color: var(--bg);
      color: var(--text);
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
    }
    .custom-header {
      background: rgba(17, 24, 39, 0.85);
      backdrop-filter: blur(8px);
      border-bottom: 1px solid var(--border);
      padding: 14px 28px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      position: sticky;
      top: 0;
      z-index: 100;
    }
    .custom-brand {
      display: flex;
      align-items: center;
      gap: 12px;
      font-weight: 700;
      font-size: 18px;
      color: var(--text);
      text-decoration: none;
    }
    .custom-brand .logo-box {
      width: 32px;
      height: 32px;
      border-radius: 8px;
      background: linear-gradient(135deg, var(--primary) 0%, #0284c7 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 16px;
    }
    .custom-brand span.hub {
      color: var(--primary);
      font-size: 12px;
      text-transform: uppercase;
      margin-left: 4px;
      letter-spacing: 0.05em;
    }
    .nav-links a {
      color: #9ca3af;
      text-decoration: none;
      font-size: 14px;
      padding: 6px 12px;
      border-radius: 6px;
      transition: all 0.15s ease;
      border: 1px solid transparent;
    }
    .nav-links a:hover {
      color: var(--text);
      background: rgba(255, 255, 255, 0.05);
      border-color: var(--border);
    }
    /* Dark Theme Adjustments for Swagger UI */
    .swagger-ui {
      max-width: 1200px;
      margin: 0 auto;
      padding: 20px;
    }
    .swagger-ui .topbar { display: none !important; }
    .swagger-ui .info { margin: 20px 0 !important; }
    .swagger-ui .info .title { color: #38bdf8 !important; }
    .swagger-ui .scheme-container { background: transparent !important; box-shadow: none !important; }
    .swagger-ui .opblock { border-radius: 8px !important; margin: 0 0 16px !important; }
    .swagger-ui code { font-family: 'JetBrains Mono', monospace !important; }
  </style>
</head>
<body>
  <header class="custom-header">
    <a href="/" class="custom-brand">
      <div class="logo-box">🏠</div>
      <span>Domus <span class="hub">REST API</span></span>
    </a>
    <div class="nav-links">
      <a href="/api/openapi.yaml" target="_blank" download="domus-openapi.yaml">Download OpenAPI YAML</a>
      <a href="/">Open App</a>
    </div>
  </header>

  <div id="swagger-ui"></div>

  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/api/openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: "StandaloneLayout"
      });
    };
  </script>
</body>
</html>
`
