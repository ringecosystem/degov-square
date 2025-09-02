{{template "layout.md" .}}

{{define "content"}}
Hello,

Your One-Time Password (OTP) is: **{{.OTP}}**

This code will expire in 10 minutes.
{{end}}
