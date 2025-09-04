package services

import (
	"log/slog"
	"strings"

	"github.com/ringecosystem/degov-apps/internal/config"
	"github.com/ringecosystem/degov-apps/types"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gorm.io/gorm"
)

var globalNotifier *NotifierService

func getNotifier() *NotifierService {
	if globalNotifier == nil {
		globalNotifier = &NotifierService{}
	}
	return globalNotifier
}

type NotifierService struct {
	db *gorm.DB
}

func NewNotifierService() *NotifierService {
	return getNotifier()
}

func (n *NotifierService) Notify(input types.NotifyInput) error {
	sendgridApiKey := config.GetString("SENDGRID_API_KEY")
	if sendgridApiKey != "" {
		if err := n.notifyUseSendGrid(input); err != nil {
			slog.Warn("Failed to send notification [sendgrid]", "error", err)
		}
	}
	return nil
}

func (n *NotifierService) notifyUseSendGrid(input types.NotifyInput) error {
	template := input.Template
	from := mail.NewEmail(config.GetString("SENDGRID_FROM_USER"), config.GetString("SENDGRID_FROM_EMAIL"))
	nameParts := strings.Split(input.To, "@")
	name := nameParts[0]
	to := mail.NewEmail(name, input.To)
	subject := template.Title
	plainTextContent := template.PlainTextContent
	htmlContent := template.RichTextContent
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(config.GetString("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		slog.Error("Failed to send notification", "error", err)
		return err
	}
	slog.Info("Notification sent successfully", "to", input.To, "status_code", response.StatusCode)
	return nil
}
