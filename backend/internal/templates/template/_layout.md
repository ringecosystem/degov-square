{{define "layout.md"}}

## DeGov.AI

---

{{block "content" .}}
This is default content, which should be overridden by a specific template.
{{end}}

---

[Home]({{.DegovSiteConfig.Home}})
[Square]({{.DegovSiteConfig.Square}})
[Docs]({{.DegovSiteConfig.Docs}})

Follow us:
{{range .DegovSiteConfig.Socials}}
[{{.Name}}]({{.Link}})
{{end}}

DeGov.AI

Want to change how you receive these emails?
You can update your subscribe preferences https://square.degov.ai/notification/subscription
{{end}}
