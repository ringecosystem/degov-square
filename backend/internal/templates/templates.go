package templates

import (
	"embed"
)

// Embed all template files, including html and md
//
//go:embed template/*.html
//go:embed template/*.md
//go:embed prompts/*.md
var TemplateFS embed.FS
