package types

type GenerateTemplateOTPInput struct {
	DegovSiteConfig DegovSiteConfig `json:"degov_site_config"`
	EmailStyle      *EmailStyle      `json:"email_style"`
	OTP             string          `json:"otp"`
	Expiration      int             `json:"expiration"`
	UserAddress     string          `json:"user_address"`
	EnsName         *string         `json:"ens_name"`
}

type TemplateOutput struct {
	Title            string `json:"title"`
	RichTextContent  string `json:"rich_text_content"`
	PlainTextContent string `json:"plain_text_content"`
}
