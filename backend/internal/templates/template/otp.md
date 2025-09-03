{{define "content"}}

Hello {{with .EnsName}}{{.}}{{else}}{{.UserAddress}}{{end}},

To confirm and link this email address to your account, please use the following One-Time Password (OTP).

Your confirmation code is:

**{{.OTP}}**

This code is valid for {{.Expiration}} minutes.

If you did not make this request, please disregard this email. No further action is required.
{{end}}
