package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/internal/config"
	"github.com/ringecosystem/degov-apps/internal/templates"
	"github.com/ringecosystem/degov-apps/types"
)

type TemplateService struct {
	daoService       *DaoService
	proposalService  *ProposalService
	daoConfigService *DaoConfigService
}

func NewTemplateService() *TemplateService {
	return &TemplateService{
		daoService:       NewDaoService(),
		proposalService:  NewProposalService(),
		daoConfigService: NewDaoConfigService(),
	}
}

type TemplateNotificationRecordData struct {
	DegovSiteConfig types.DegovSiteConfig      `json:"degov_site_config"`
	DaoConfig       *types.DaoConfig           `json:"dao_config"`
	Dao             *gqlmodels.Dao             `json:"dao"`
	Proposal        *dbmodels.ProposalTracking `json:"proposal"`
	Vote            *internal.VoteCast         `json:"vote,omitempty"`
	PayloadData     map[string]interface{}     `json:"payload_data"`
	EventID         string                     `json:"event_id"`
	UserID          string                     `json:"user_id"`
	UserAddress     string                     `json:"user_address"`
}

type TemplateOTPData struct {
	DegovSiteConfig types.DegovSiteConfig `json:"degov_site_config"`
	OTP             string                `json:"otp"`
}

// parsePayload attempts to parse the payload as JSON, falls back to string if failed
func (s *TemplateService) parsePayload(payload *string) map[string]interface{} {
	result := make(map[string]interface{})

	if payload == nil || *payload == "" {
		return result
	}

	// Try to parse as JSON first
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(*payload), &jsonData); err == nil {
		return jsonData
	}

	// If JSON parsing fails, treat as plain string
	result["__raw"] = *payload
	return result
}

// getTemplateFileName returns the template file name based on notification type
func (s *TemplateService) getTemplateFileName(notificationType dbmodels.SubscribeFeatureName) string {
	switch notificationType {
	case dbmodels.SubscribeFeatureProposalNew:
		return "proposal_new.html"
	case dbmodels.SubscribeFeatureProposalStateChanged:
		return "proposal_state_changed.html"
	case dbmodels.SubscribeFeatureVoteEnd:
		return "vote_end.html"
	case dbmodels.SubscribeFeatureVoteEmitted:
		return "vote_emitted.html"
	default:
		return "proposal_new.html" // fallback
	}
}

func (s *TemplateService) GenerateTemplateByNotificationRecord(record *dbmodels.NotificationRecord) (string, error) {
	// Get DAO information
	dao, err := s.daoService.Inspect(types.BasicInput[string]{
		User:  nil,
		Input: record.DaoCode,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get DAO info: %w", err)
	}

	// Get proposal information
	proposal, err := s.proposalService.InspectProposal(types.InpspectProposalInput{
		DaoCode:    record.DaoCode,
		ProposalID: record.ProposalID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get proposal info: %w", err)
	}

	daoConfig, err := s.daoConfigService.StandardConfig(dao.Code)
	if err != nil {
		return "", fmt.Errorf("failed to get DAO config info: %w", err)
	}

	var vote *internal.VoteCast
	if record.VoteID != nil {
		degovIndexer := internal.NewDegovIndexer(daoConfig.Indexer.Endpoint)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		voteById, err := degovIndexer.QueryVote(ctx, *record.VoteID)
		cancel()

		if err != nil {
			return "", fmt.Errorf("failed to get vote info: %w", err)
		}

		vote = voteById
	}

	// Parse payload data
	payloadData := s.parsePayload(record.Payload)

	templateData := TemplateNotificationRecordData{
		DegovSiteConfig: config.GetDegovSiteConfig(),
		DaoConfig:       daoConfig,
		Dao:             dao,
		Proposal:        proposal,
		Vote:            vote,
		PayloadData:     payloadData,
		EventID:         record.EventID,
		UserID:          record.UserID,
		UserAddress:     record.UserAddress,
	}

	// Get template file name based on notification type
	templateFileName := s.getTemplateFileName(record.Type)

	// Get template file path for embedded filesystem
	templatePath := "template/" + templateFileName

	// Read template content from embedded filesystem
	templateContent, err := templates.TemplateFS.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded template file: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New(templateFileName).Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

func (s *TemplateService) GenerateTemplateOTP(input types.GenerateTemplateOTPInput) (string, error) {
	templateContent, err := templates.TemplateFS.ReadFile("template/otp.html")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded template file: %w", err)
	}

	// Parse and execute template
	tmpl, err := template.New("otp").Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	templateData := TemplateOTPData{
		DegovSiteConfig: config.GetDegovSiteConfig(),
		OTP:             input.OTP,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
