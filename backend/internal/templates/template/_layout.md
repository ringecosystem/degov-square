{{define "layout.md"}}

## DeGov.AI

---

{{block "content" .}}
This is default content, which should be overridden by a specific template.
{{end}}

---

{{if and .DaoConfig .DaoConfig.Name}}{{.DaoConfig.Name}} - {{end}}DeGov.AI

Want to change how you receive these emails?
You can update your subscribe preferences https://apps.degiv.ai/subscribe/preference
{{end}}
