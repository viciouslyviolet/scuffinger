package server

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed openapi.yaml
var openapiSpec []byte

// RegisterDocs registers the API documentation endpoints:
//   - GET /docs          → Swagger UI
//   - GET /docs/openapi.yaml → raw OpenAPI spec
func RegisterDocs(r *gin.Engine) {
	r.GET("/docs", swaggerUIHandler)
	r.GET("/docs/openapi.yaml", openapiSpecHandler)
}

// openapiSpecHandler serves the embedded OpenAPI YAML spec.
func openapiSpecHandler(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml", openapiSpec)
}

// swaggerUIHandler serves an HTML page that loads Swagger UI from a CDN.
func swaggerUIHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
}

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Scuffinger — API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.24.1/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *::before, *::after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
    .topbar { display: none !important; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.24.1/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/docs/openapi.yaml",
      dom_id: "#swagger-ui",
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset,
      ],
      layout: "BaseLayout",
    });
  </script>
</body>
</html>`

