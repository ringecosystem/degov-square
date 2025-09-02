package types

type GenerateTemplateOTPInput struct {
	OTP string `json:"otp"`
}

type TemplateOutput struct {
	Title            string `json:"title"`
	RichTextContent  string `json:"rich_text_content"`
	PlainTextContent string `json:"plain_text_content"`
}
