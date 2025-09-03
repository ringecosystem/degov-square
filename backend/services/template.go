package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	tplHtml "html/template"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	tplText "text/template"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal"
	"github.com/ringecosystem/degov-apps/internal/config"
	"github.com/ringecosystem/degov-apps/internal/templates"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"github.com/ringecosystem/degov-apps/types"
)

type TemplateService struct {
	daoService       *DaoService
	proposalService  *ProposalService
	daoConfigService *DaoConfigService
	htmlTemplates    *tplHtml.Template
	textTemplates    *tplText.Template
	userService      *UserService
}

func NewTemplateService() *TemplateService {
	funcMap := tplText.FuncMap{
		"formatDate": formatDate,
	}
	htmlTmpls := tplHtml.Must(tplHtml.New("").Funcs(funcMap).ParseFS(
		templates.TemplateFS,
		"template/*.html",
	))

	textTmpls := tplText.Must(tplText.New("").Funcs(funcMap).ParseFS(
		templates.TemplateFS,
		"template/*.md",
	))
	return &TemplateService{
		daoService:       NewDaoService(),
		proposalService:  NewProposalService(),
		daoConfigService: NewDaoConfigService(),
		htmlTemplates:    htmlTmpls,
		textTemplates:    textTmpls,
		userService:      NewUserService(),
	}
}

type templateNotificationRecordData struct {
	DegovSiteConfig types.DegovSiteConfig  `json:"degov_site_config"`
	EmailStyle      *types.EmailStyle      `json:"email_style"`
	Title           *string                `json:"title"`
	DaoConfig       *types.DaoConfig       `json:"dao_config"`
	Dao             *gqlmodels.Dao         `json:"dao"`
	Proposal        *emailProposalInfo     `json:"proposal"`
	Vote            *emailVoteInfo         `json:"vote,omitempty"`
	PayloadData     map[string]interface{} `json:"payload_data"`
	EventID         string                 `json:"event_id"`
	UserID          string                 `json:"user_id"`
	UserAddress     string                 `json:"user_address"`
	EnsName         *string                `json:"ens_name"`
}

type emailProposalInfo struct {
	ProposalDb                  *dbmodels.ProposalTracking `json:"proposal_db"`
	ProposalIndexer             *internal.Proposal         `json:"proposal_indexer"`
	ProposalDescriptionMarkdown *string                    `json:"proposal_description_markdown"`
	ProposalDescriptionHtml     *string                    `json:"proposal_description_html"`
	ProposerEnsName             *string                    `json:"proposer_ens_name"`
	TweetLink                   *string                    `json:"tweet_link"`
}

type emailVoteInfo struct {
	VoteIndexer *internal.VoteCast `json:"vote_indexer"`
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

func (s *TemplateService) getTemplateFileName(notificationType dbmodels.SubscribeFeatureName, mode string) string {
	switch notificationType {
	case dbmodels.SubscribeFeatureProposalNew:
		return "proposal_new." + mode
	case dbmodels.SubscribeFeatureProposalStateChanged:
		return "proposal_state_changed." + mode
	case dbmodels.SubscribeFeatureVoteEnd:
		return "vote_end." + mode
	case dbmodels.SubscribeFeatureVoteEmitted:
		return "vote_emitted." + mode
	default:
		return "unknown." + mode // fallback
	}
}

func (s *TemplateService) GenerateTemplateByNotificationRecord(record *dbmodels.NotificationRecord) (*types.TemplateOutput, error) {
	// Get DAO information
	dao, err := s.daoService.Inspect(types.BasicInput[string]{
		User:  nil,
		Input: record.DaoCode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get DAO info: %w", err)
	}

	// Get proposal information
	proposal, err := s.proposalService.InspectProposal(types.InpspectProposalInput{
		DaoCode:    record.DaoCode,
		ProposalID: record.ProposalID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get proposal info: %w", err)
	}

	daoConfig, err := s.daoConfigService.StandardConfig(dao.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get DAO config info: %w", err)
	}

	degovIndexer := internal.NewDegovIndexer(daoConfig.Indexer.Endpoint)

	var vote *internal.VoteCast
	if record.VoteID != nil {
		voteById, err := degovIndexer.QueryVote(*record.VoteID)

		if err != nil {
			return nil, fmt.Errorf("failed to get vote info: %w", err)
		}

		vote = voteById
	}

	// Parse payload data
	payloadData := s.parsePayload(record.Payload)
	title := "New notification from DeGov.AI"
	emailProposal := emailProposalInfo{
		ProposalDb: proposal,
		// ProposalIndexer:             proposalIndexer,
		// FormatedProposalDescription: proposal.FormatedDescription,
	}

	if record.Type == dbmodels.SubscribeFeatureProposalNew || record.Type == dbmodels.SubscribeFeatureVoteEnd {
		proposalIndexer, err := degovIndexer.InspectProposal(proposal.ProposalID)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect full proposal: %w", err)
		}
		emailProposal.ProposalIndexer = proposalIndexer
	}

	switch record.Type {
	case dbmodels.SubscribeFeatureProposalNew:
		title = fmt.Sprintf("[%s] New Proposal: %s", dao.Name, proposal.Title)
		proposalIndexer := emailProposal.ProposalIndexer
		if !config.GetDegovSiteConfig().EmailProposalIncludeDescription {
			proposalDescriptionHtml := mdToHTML([]byte(proposalIndexer.Description))
			emailProposal.ProposalDescriptionHtml = &proposalDescriptionHtml
			proposalDescriptionMarkdown, err := htmltomarkdown.ConvertString(proposalDescriptionHtml)
			if err != nil {
				slog.Warn("failed to convert html to markdown", "error", err)
			} else {
				emailProposal.ProposalDescriptionMarkdown = &proposalDescriptionMarkdown
			}
		}

		ensName, err := s.userService.GetENSName(proposalIndexer.Proposer)
		if err != nil {
			slog.Warn("failed to query ens name for user", "user_address", proposalIndexer.Proposer, "error", err)
		} else {
			emailProposal.ProposerEnsName = ensName
		}
		degovAgent := internal.NewDegovAgent()
		agentVote, err := degovAgent.QueryVote(int(dao.ChainID), proposal.ProposalID)
		if err != nil {
			slog.Warn("[degov-agent] failed to query vote", "error", err)
		} else {
			tweetLink := fmt.Sprintf("https://x.com/%s/status/%s", agentVote.TwitterUser.Username, agentVote.ID)
			emailProposal.TweetLink = &tweetLink
		}
		break
	case dbmodels.SubscribeFeatureProposalStateChanged:
		title = fmt.Sprintf("[%s] Proposal Status Update: %s", dao.Name, proposal.Title)
		break
	case dbmodels.SubscribeFeatureVoteEnd:
		title = fmt.Sprintf("[%s] Vote End Reminder: %s", dao.Name, proposal.Title)
		break
	case dbmodels.SubscribeFeatureVoteEmitted:
		title = fmt.Sprintf("[%s] Vote Emitted: %s", dao.Name, proposal.Title)
		break
	}

	ensName, err := s.userService.GetENSName(record.UserAddress)
	if err != nil {
		slog.Warn("failed to query ens name for user", "user_address", record.UserAddress, "error", err)
	}

	degovSiteConfig := config.GetDegovSiteConfig()

	emailStyle := config.GetEmailStyle()
	emailStyle.ContainerMaxWidth = "85%"

	templateData := templateNotificationRecordData{
		DegovSiteConfig: degovSiteConfig,
		EmailStyle:      &emailStyle,
		Title:           &title,
		DaoConfig:       daoConfig,
		Dao:             dao,
		Proposal:        &emailProposal,
		Vote:            vote,
		PayloadData:     payloadData,
		EventID:         record.EventID,
		UserID:          record.UserID,
		UserAddress:     record.UserAddress,
		EnsName:         ensName,
	}

	richTemplateFileName := s.getTemplateFileName(record.Type, "html")
	plainTemplateFileName := s.getTemplateFileName(record.Type, "md")

	richText, err := s.renderTemplate(richTemplateFileName, templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render rich text template %s: %w", richTemplateFileName, err)
	}
	palinText, err := s.renderTemplate(plainTemplateFileName, templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render plain text template %s: %w", plainTemplateFileName, err)
	}

	return &types.TemplateOutput{
		Title:            utils.TruncateText(title, 80),
		RichTextContent:  richText,
		PlainTextContent: palinText,
	}, nil
}

func (s *TemplateService) renderTemplate(templateName string, data interface{}) (string, error) {
	var finData interface{}
	templateData, serr := structToMap(data)
	if serr != nil {
		if m, ok := data.(map[string]interface{}); ok {
			finData = m
		} else {
			finData = data
		}
	} else {
		finData = templateData
	}

	var buf bytes.Buffer
	var err error

	if strings.HasSuffix(templateName, ".html") {
		err = s.htmlTemplates.ExecuteTemplate(&buf, templateName, finData)
	} else if strings.HasSuffix(templateName, ".md") {
		err = s.textTemplates.ExecuteTemplate(&buf, templateName, finData)
	} else {
		return "", fmt.Errorf("unsupported template type: %s", templateName)
	}

	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	renderedContent := buf.Bytes()
	if config.GetAppEnv().IsDevelopment() {
		if outputBasePath := config.GetString("DEBUG_TEMPLATE_OUTPUT_PATH"); outputBasePath != "" {
			go writeDebugTemplateFile(outputBasePath, templateName, renderedContent)
		}
	}

	return string(renderedContent), nil
}

func (s *TemplateService) GenerateTemplateOTP(input types.GenerateTemplateOTPInput) (*types.TemplateOutput, error) {
	if input.EmailStyle == nil {
		emailStyle := config.GetEmailStyle()
		input.EmailStyle = &emailStyle
	}
	richText, err := s.renderTemplate("otp.html", input)
	if err != nil {
		slog.Error("failed to render OTP html template", "err", err)
		return nil, err
	}
	plainText, err := s.renderTemplate("otp.md", input)
	if err != nil {
		slog.Error("failed to render OTP md template", "err", err)
		return nil, err
	}

	return &types.TemplateOutput{
		Title:            "[DeGov] Email Verification",
		RichTextContent:  richText,
		PlainTextContent: plainText,
	}, nil

}

func structToMap(data interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	val := reflect.ValueOf(data)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("data is not a struct, but a %s", val.Kind())
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if field.IsExported() {
			result[field.Name] = val.Field(i).Interface()
		}
	}

	return result, nil
}

func writeDebugTemplateFile(outputBasePath, templateName string, content []byte) {
	if err := os.MkdirAll(outputBasePath, 0755); err != nil {
		slog.Warn("WARN: Failed to create debug template directory %s: %v", outputBasePath, err)
		return
	}

	ext := filepath.Ext(templateName)

	sanitizedTemplateName := strings.ReplaceAll(templateName, "/", "_")
	sanitizedBaseName := strings.TrimSuffix(sanitizedTemplateName, ext)

	var filePath string
	for i := 1; i < 10000000; i++ {
		fileName := fmt.Sprintf("%s_%07d.local.%s", sanitizedBaseName, i, ext)
		filePath = filepath.Join(outputBasePath, fileName)

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			break
		}
	}

	if filePath == "" {
		slog.Warn("WARN: Could not find an available debug file name for %s in %s", templateName, outputBasePath)
		return
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		slog.Warn("WARN: Failed to write debug template to %s: %v", filePath, err)
	} else {
		slog.Info("INFO: Debug template written", "filePath", filePath)
	}
}

func mdToHTML(md []byte) string {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	maybeUnsafeHTML := markdown.Render(doc, renderer)
	return string(bluemonday.UGCPolicy().SanitizeBytes(maybeUnsafeHTML))
}

// formatDate formats a Unix timestamp string into the "Month Day, Year at Hour:Minute PM Timezone" layout.
func formatDate(timestampStr string) string {
	i, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		slog.Warn("Could not parse timestamp string", "timestampStr", timestampStr, "error", err)
		return timestampStr
	}

	// Create a time.Time object from the Unix timestamp (seconds).
	t := time.Unix(i, 0)

	// 1. Convert the time to the UTC timezone.
	t = t.UTC()

	// 2. Use the new format layout string.
	//    Based on Go's reference time: Mon Jan 2 15:04:05 MST 2006
	//    "January" -> Full month name (e.g., "September")
	//    "2"       -> Day of the month without leading zero (e.g., "2")
	//    "2006"    -> Four-digit year (e.g., "2025")
	//    "at"      -> The literal string "at"
	//    "3"       -> Hour in 12-hour format without leading zero (e.g., "4")
	//    "04"      -> Minute with leading zero (e.g., "04")
	//    "PM"      -> AM/PM marker (e.g., "PM")
	//    "MST"     -> Timezone abbreviation (will display "UTC" since we converted to UTC)
	return t.Format("January 2, 2006 at 3:04 PM MST")
}
