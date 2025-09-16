package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	tplHtml "html/template"
	"log/slog"
	"math/big"
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
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/internal/templates"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/ringecosystem/degov-square/types"
	"github.com/vanng822/go-premailer/premailer"
)

type TemplateService struct {
	daoService       *DaoService
	proposalService  *ProposalService
	daoConfigService *DaoConfigService
	htmlTemplates    map[string]*tplHtml.Template
	textTemplates    map[string]*tplText.Template
	userService      *UserService
}

func NewTemplateService() *TemplateService {
	funcMap := tplText.FuncMap{
		"formatDate":               utils.FormatDate,
		"formatLargeNumber":        utils.FormatLargeNumber,
		"formatDecimal":            utils.FormatDecimal,
		"formatPercent":            utils.FormatPercent,
		"formatDurationShort":      utils.FormatDurationShort,
		"formatBigIntWithDecimals": utils.FormatBigIntWithDecimals,
		"formatAsMdQuote":          utils.FormatAsMdQuote,
	}

	files, err := templates.TemplateFS.ReadDir("template")
	if err != nil {
		panic(err)
	}

	htmlTmpls := make(map[string]*tplHtml.Template)
	for _, file := range files {
		fileName := file.Name()
		if file.IsDir() || !strings.HasSuffix(fileName, ".html") || fileName == "_layout.html" {
			continue
		}

		tmpl := tplHtml.Must(tplHtml.New(fileName).Funcs(funcMap).ParseFS(
			templates.TemplateFS,
			"template/_layout.html",
			"template/"+fileName,
		))

		htmlTmpls[fileName] = tmpl
	}

	textTmpls := make(map[string]*tplText.Template)
	for _, file := range files {
		fileName := file.Name()
		if file.IsDir() || !strings.HasSuffix(fileName, ".md") || fileName == "_layout.md" {
			continue
		}

		tmpl := tplText.Must(tplText.New(fileName).Funcs(funcMap).ParseFS(
			templates.TemplateFS,
			"template/_layout.md",
			"template/"+fileName,
		))

		textTmpls[fileName] = tmpl
	}
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
	ProposalDescriptionHtml     *tplHtml.HTML              `json:"proposal_description_html"`
	ProposerEnsName             *string                    `json:"proposer_ens_name"`
	TweetLink                   *string                    `json:"tweet_link"`
}

type emailVoteInfo struct {
	VoteIndexer    *internal.VoteCast `json:"vote_indexer"`
	TotalVotePower string             `json:"total_vote_power"`
	PercentFor     float64            `json:"percent_for"`
	PercentAgainst float64            `json:"percent_against"`
	PercentAbstain float64            `json:"percent_abstain"`
	PercentQuorum  float64            `json:"percent_quorum"`
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
	proposal, err := s.proposalService.InspectProposal(types.InspectProposalInput{
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

	var emailVote emailVoteInfo

	// Parse payload data
	payloadData := s.parsePayload(record.Payload)
	title := "New notification from DeGov.AI"
	emailProposal := emailProposalInfo{
		ProposalDb: proposal,
	}

	proposalIndexer, err := degovIndexer.InspectProposal(proposal.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect full proposal: %w", err)
	}
	emailProposal.ProposalIndexer = proposalIndexer

	decimalsInt, err := strconv.Atoi(proposalIndexer.Decimals)
	if err != nil {
		slog.Warn("failed to parse decimals to int", "decimals", proposalIndexer.Decimals, "error", err)
		payloadData["DecimalsInt"] = 1
	} else {
		payloadData["DecimalsInt"] = decimalsInt
	}

	if record.Type == dbmodels.SubscribeFeatureVoteEmitted {
		voteIndexer, err := degovIndexer.QueryVote(*record.VoteID)
		if err != nil {
			return nil, fmt.Errorf("failed to get vote info: %w", err)
		}
		emailVote.VoteIndexer = voteIndexer
	}

	if record.Type == dbmodels.SubscribeFeatureVoteEnd {
		voteIndexer, err := degovIndexer.QueryVoteByVoter(proposal.ProposalID, record.UserAddress)
		if err != nil {
			slog.Warn("failed to get vote for this user", "user_address", record.UserAddress, "error", err)
		} else {
			emailVote.VoteIndexer = voteIndexer
		}
	}

	switch record.Type {
	case dbmodels.SubscribeFeatureProposalNew:
		title = fmt.Sprintf("[%s] New Proposal: %s", dao.Name, proposal.Title)
		if config.GetDegovSiteConfig().EmailProposalIncludeDescription {
			proposalDescriptionHtml := mdToHTML([]byte(proposalIndexer.Description))
			proposalDescriptionHtmlTplHtmlContent := tplHtml.HTML(proposalDescriptionHtml)
			emailProposal.ProposalDescriptionHtml = &proposalDescriptionHtmlTplHtmlContent

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
	case dbmodels.SubscribeFeatureProposalStateChanged:
		title = fmt.Sprintf("[%s] Proposal Status Update: %s", dao.Name, proposal.Title)
	case dbmodels.SubscribeFeatureVoteEnd:
		title = fmt.Sprintf("[%s] Vote End Reminder: %s", dao.Name, proposal.Title)
		emailVote.TotalVotePower = calculateTotalVotePower(proposalIndexer)
		if proposalIndexer.MetricsVotesWeightForSum != nil {
			emailVote.PercentFor = utils.CalculateBigIntRatioPercentage(*proposalIndexer.MetricsVotesWeightForSum, emailVote.TotalVotePower)
		}
		if proposalIndexer.MetricsVotesWeightAgainstSum != nil {
			emailVote.PercentAgainst = utils.CalculateBigIntRatioPercentage(*proposalIndexer.MetricsVotesWeightAgainstSum, emailVote.TotalVotePower)
		}
		if proposalIndexer.MetricsVotesWeightAbstainSum != nil {
			emailVote.PercentAbstain = utils.CalculateBigIntRatioPercentage(*proposalIndexer.MetricsVotesWeightAbstainSum, emailVote.TotalVotePower)
		}
		emailVote.PercentQuorum = utils.CalculateBigIntRatioPercentage(emailVote.TotalVotePower, proposalIndexer.Quorum)
		voteEndTime, err := utils.ParseTimestamp(proposalIndexer.VoteEndTimestamp)
		if err != nil {
			slog.Warn("failed to parse vote end timestamp", "timestamp", proposalIndexer.VoteEndTimestamp, "error", err)
		} else {
			payloadData["TimeRemaining"] = utils.FormatDurationShort(time.Until(voteEndTime))
		}
	case dbmodels.SubscribeFeatureVoteEmitted:
		title = fmt.Sprintf("[%s] Vote Emitted: %s", dao.Name, proposal.Title)
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
		Vote:            &emailVote,
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
	plainText, err := s.renderTemplate(plainTemplateFileName, templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render plain text template %s: %w", plainTemplateFileName, err)
	}

	return &types.TemplateOutput{
		Title:            utils.TruncateText(title, 80),
		RichTextContent:  richText,
		PlainTextContent: plainText,
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

	htmlMode := strings.HasSuffix(templateName, ".html")
	if htmlMode {
		tmpl, ok := s.htmlTemplates[templateName]
		if !ok {
			return "", fmt.Errorf("html template %s not found", templateName)
		}

		err = tmpl.ExecuteTemplate(&buf, "layout.html", finData)
	} else if strings.HasSuffix(templateName, ".md") {
		tmpl, ok := s.textTemplates[templateName]
		if !ok {
			return "", fmt.Errorf("md template %s not found", templateName)
		}

		err = tmpl.ExecuteTemplate(&buf, "layout.md", finData)
	} else {
		return "", fmt.Errorf("unsupported template type: %s", templateName)
	}

	if err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	renderedBytes := buf.Bytes()
	renderedText := string(renderedBytes)

	if htmlMode {
		options := premailer.NewOptions()
		options.RemoveClasses = true
		prem, err := premailer.NewPremailerFromString(renderedText, options)
		if err != nil {
			return "nil", fmt.Errorf("failed to create premailer from rich text: %w", err)
		}
		inlinesHtml, err := prem.Transform()
		if err != nil {
			return "nil", fmt.Errorf("failed to transform rich text: %w", err)
		}
		renderedText = inlinesHtml
	}

	if config.GetAppEnv().IsDevelopment() {
		if outputBasePath := config.GetString("DEBUG_TEMPLATE_OUTPUT_PATH"); outputBasePath != "" {
			go writeDebugTemplateFile(outputBasePath, templateName, []byte(renderedText))
		}
	}

	return renderedText, nil
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
		fileName := fmt.Sprintf("%s_%07d.local%s", sanitizedBaseName, i, ext)
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

func calculateTotalVotePower(proposal *internal.Proposal) string {
	total := new(big.Int)

	fields := []*string{
		proposal.MetricsVotesWeightForSum,
		proposal.MetricsVotesWeightAgainstSum,
		proposal.MetricsVotesWeightAbstainSum,
	}

	for _, fieldPtr := range fields {
		if fieldPtr == nil || *fieldPtr == "" {
			continue
		}

		currentVal := new(big.Int)

		_, ok := currentVal.SetString(*fieldPtr, 10)
		if !ok {
			slog.Warn("Could not parse bigint string, skipping", "value", *fieldPtr)
			continue
		}

		total.Add(total, currentVal)
	}

	return total.String()
}
