package templates

import (
	"embed"
)

// Embed all template files
//
//go:embed template/*.html
var TemplateFS embed.FS
