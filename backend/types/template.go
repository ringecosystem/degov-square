package types

type GenerateTemplateOTPInput struct {
	OTP string `json:"otp"`
}

type TemplateOutput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
