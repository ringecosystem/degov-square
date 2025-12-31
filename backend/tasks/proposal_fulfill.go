package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/services"
	"github.com/ringecosystem/degov-square/types"
)

// ProposalFulfillTask handles AI-based voting on proposals
type ProposalFulfillTask struct {
	daoService       *services.DaoService
	daoConfigService *services.DaoConfigService
	proposalService  *services.ProposalService
	openRouterClient *internal.OpenRouterClient
	supportedDAOs    []string // nil means all DAOs
}

// NewProposalFulfillTask creates a new proposal fulfill task
func NewProposalFulfillTask() *ProposalFulfillTask {
	cfg := config.GetConfig()
	supportedDAOs := cfg.GetTaskProposalFulfillDAOs()

	if len(supportedDAOs) > 0 {
		slog.Info("[proposal-fulfill] Task initialized with specific DAOs", "daos", supportedDAOs)
	} else {
		slog.Info("[proposal-fulfill] Task initialized for all DAOs")
	}

	return &ProposalFulfillTask{
		daoService:       services.NewDaoService(),
		daoConfigService: services.NewDaoConfigService(),
		proposalService:  services.NewProposalService(),
		openRouterClient: internal.NewOpenRouterClient(),
		supportedDAOs:    supportedDAOs,
	}
}

// Name returns the task name
func (t *ProposalFulfillTask) Name() string {
	return "proposal-fulfill"
}

// Execute performs the proposal fulfill task
func (t *ProposalFulfillTask) Execute() error {
	return t.fulfillProposals()
}

// fulfillProposals processes unfulfilled proposals
func (t *ProposalFulfillTask) fulfillProposals() error {
	// Get unfulfilled proposals (filtered by supported DAOs if configured)
	proposals, err := t.proposalService.ListUnfulfilledProposals(t.supportedDAOs)
	if err != nil {
		slog.Error("Failed to list unfulfilled proposals", "error", err)
		return err
	}

	if len(proposals) == 0 {
		slog.Info("[proposal-fulfill] No unfulfilled proposals found, skipping task")
		return nil
	}

	slog.Info("[proposal-fulfill] Found unfulfilled proposals", "count", len(proposals))

	// Filter proposals that are ready (past midpoint)
	readyProposals, err := t.filterReadyProposals(proposals)
	if err != nil {
		slog.Error("[proposal-fulfill] Failed to filter ready proposals", "error", err)
		return err
	}

	if len(readyProposals) == 0 {
		slog.Info("[proposal-fulfill] No proposals past midpoint yet, skipping this cycle")
		return nil
	}

	// Process each ready proposal
	for _, item := range readyProposals {
		if err := t.fulfillProposal(item.proposal, item.daoConfig); err != nil {
			slog.Error("[proposal-fulfill] Failed to fulfill proposal",
				"proposal_id", item.proposal.ProposalID,
				"dao_code", item.proposal.DaoCode,
				"error", err)

			// Update error tracking
			if updateErr := t.proposalService.UpdateFulfillError(
				item.proposal.ProposalID,
				item.proposal.DaoCode,
				err.Error(),
			); updateErr != nil {
				slog.Error("[proposal-fulfill] Failed to update fulfill error",
					"proposal_id", item.proposal.ProposalID,
					"error", updateErr)
			}
			continue
		}

		slog.Info("[proposal-fulfill] Successfully fulfilled proposal",
			"proposal_id", item.proposal.ProposalID,
			"dao_code", item.proposal.DaoCode)
	}

	return nil
}

// readyProposalItem holds a proposal with its DAO config
type readyProposalItem struct {
	proposal  *dbmodels.ProposalTracking
	daoConfig *types.DaoConfig
}

// filterReadyProposals filters proposals that are past their voting midpoint but not yet expired
func (t *ProposalFulfillTask) filterReadyProposals(proposals []*dbmodels.ProposalTracking) ([]readyProposalItem, error) {
	var ready []readyProposalItem

	// Get agent address for delegation check
	agentAddress := internal.GetAgentAddress()
	if agentAddress == "" {
		slog.Warn("[proposal-fulfill] Agent address not configured, skipping delegation check")
	}

	for _, proposal := range proposals {
		// Get DAO config
		daoConfig, err := t.daoConfigService.StandardConfig(proposal.DaoCode)
		if err != nil {
			slog.Warn("[proposal-fulfill] Failed to get DAO config",
				"dao_code", proposal.DaoCode,
				"error", err)
			continue
		}

		// Create indexer to query proposal details
		indexer := internal.NewDegovIndexer(daoConfig.Indexer.Endpoint)

		// Check if agent has delegators (other than itself)
		if agentAddress != "" {
			ctx := context.Background()
			hasDelegators, err := indexer.HasDelegatorsOtherThanSelf(ctx, agentAddress)
			if err != nil {
				slog.Warn("[proposal-fulfill] Failed to check delegators",
					"proposal_id", proposal.ProposalID,
					"agent_address", agentAddress,
					"error", err)
				// Continue with other checks, don't skip due to query error
			} else if !hasDelegators {
				slog.Info("[proposal-fulfill] No delegators found for agent, marking as skipped",
					"proposal_id", proposal.ProposalID,
					"dao_code", proposal.DaoCode,
					"agent_address", agentAddress)

				// Mark as skipped - no delegators
				if updateErr := t.proposalService.MarkFulfillNoDelegators(
					proposal.ProposalID,
					proposal.DaoCode,
				); updateErr != nil {
					slog.Error("[proposal-fulfill] Failed to mark proposal as no delegators",
						"proposal_id", proposal.ProposalID,
						"error", updateErr)
				}
				continue
			}
		}

		// Query proposal to get vote window
		proposalData, err := indexer.InspectProposal(proposal.ProposalID)
		if err != nil {
			slog.Warn("[proposal-fulfill] Failed to query proposal",
				"proposal_id", proposal.ProposalID,
				"error", err)
			continue
		}

		// Check if past midpoint but not expired
		if proposalData != nil && proposalData.VoteStart != "" && proposalData.VoteEnd != "" {
			voteStart, err1 := parseInt64(proposalData.VoteStart)
			voteEnd, err2 := parseInt64(proposalData.VoteEnd)

			if err1 == nil && err2 == nil && voteEnd > voteStart {
				midpoint := voteStart + (voteEnd-voteStart)/2
				nowSeconds := time.Now().Unix()
				voteEndSeconds := voteEnd

				// Check clock mode - if timestamp mode, values are in milliseconds
				if proposalData.ClockMode == "mode=timestamp" {
					midpoint = midpoint / 1000
					voteEndSeconds = voteEnd / 1000
				}

				// Check if voting has ended - mark as expired and skip
				if nowSeconds > voteEndSeconds {
					slog.Info("[proposal-fulfill] Proposal voting has ended, marking as expired",
						"proposal_id", proposal.ProposalID,
						"dao_code", proposal.DaoCode)

					// Mark as expired so it won't be queried again
					if updateErr := t.proposalService.MarkFulfillExpired(
						proposal.ProposalID,
						proposal.DaoCode,
					); updateErr != nil {
						slog.Error("[proposal-fulfill] Failed to mark proposal as expired",
							"proposal_id", proposal.ProposalID,
							"error", updateErr)
					}
					continue
				}

				// Check if not yet at midpoint
				if nowSeconds < midpoint {
					waitSeconds := midpoint - nowSeconds
					waitMinutes := float64(waitSeconds) / 60
					slog.Info("[proposal-fulfill] Proposal not past midpoint yet, deferring",
						"proposal_id", proposal.ProposalID,
						"wait_minutes", fmt.Sprintf("%.1f", waitMinutes))
					continue
				}
			} else {
				slog.Warn("[proposal-fulfill] Invalid vote window for proposal, proceeding",
					"proposal_id", proposal.ProposalID,
					"vote_start", proposalData.VoteStart,
					"vote_end", proposalData.VoteEnd)
			}
		} else {
			slog.Warn("[proposal-fulfill] Missing vote window for proposal, proceeding",
				"proposal_id", proposal.ProposalID)
		}

		ready = append(ready, readyProposalItem{
			proposal:  proposal,
			daoConfig: daoConfig,
		})
	}

	return ready, nil
}

// fulfillProposal processes a single proposal with AI analysis and voting
func (t *ProposalFulfillTask) fulfillProposal(proposal *dbmodels.ProposalTracking, daoConfig *types.DaoConfig) error {
	// Create indexer
	indexer := internal.NewDegovIndexer(daoConfig.Indexer.Endpoint)

	// Query on-chain votes
	ctx := context.Background()
	voteCasts, err := t.queryAllVotes(ctx, indexer, proposal.ProposalID)
	if err != nil {
		return fmt.Errorf("failed to query votes: %w", err)
	}

	slog.Info("[proposal-fulfill] Queried votes for proposal",
		"proposal_id", proposal.ProposalID,
		"vote_count", len(voteCasts))

	// Convert votes to AI format
	voteCastInfos := make([]internal.VoteCastInfo, len(voteCasts))
	for i, vote := range voteCasts {
		timestamp := time.Now()
		if ts, err := parseInt64(vote.BlockTimestamp); err == nil {
			timestamp = time.Unix(ts/1000, (ts%1000)*1000000)
		}

		voteCastInfos[i] = internal.VoteCastInfo{
			Support:        internal.VoteSupportText(vote.Support),
			Reason:         vote.Reason,
			Weight:         vote.Weight,
			BlockTimestamp: timestamp,
		}
	}

	// Analyze votes with AI
	var analysisResult *internal.AnalysisResult
	for attempt := 0; attempt < 3; attempt++ {
		analysisResult, err = t.openRouterClient.AnalyzeVotes(voteCastInfos)
		if err == nil {
			break
		}

		slog.Warn("[proposal-fulfill] AI analysis attempt failed, retrying",
			"proposal_id", proposal.ProposalID,
			"attempt", attempt+1,
			"error", err)

		if attempt < 2 {
			time.Sleep(3 * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to generate AI response after 3 attempts: %w", err)
	}

	// Get RPC URL
	rpcURL := internal.GetRPCURL(daoConfig.Chain.RPCs, daoConfig.Chain.ID)
	if rpcURL == "" {
		return fmt.Errorf("no RPC URL available for chain %d", daoConfig.Chain.ID)
	}

	// Create voter client
	voter, err := internal.NewGovernorVoter(rpcURL, daoConfig.Chain.ID)
	if err != nil {
		return fmt.Errorf("failed to create governor voter: %w", err)
	}
	defer voter.Close()

	// Cast vote
	support := internal.VoteSupportToNumber(analysisResult.FinalResult)
	txHash, err := voter.CastVoteWithReason(
		ctx,
		daoConfig.Contracts.Governor,
		proposal.ProposalID,
		support,
		analysisResult.ReasoningLite,
	)
	if err != nil {
		return fmt.Errorf("failed to cast vote (tx: %s): %w", txHash, err)
	}

	slog.Info("[proposal-fulfill] Vote cast successfully",
		"proposal_id", proposal.ProposalID,
		"tx_hash", txHash,
		"support", analysisResult.FinalResult,
		"confidence", analysisResult.Confidence)

	// Save fulfilled explain
	fulfilledExplain := map[string]interface{}{
		"input": map[string]interface{}{
			"voteCasts": voteCastInfos,
		},
		"output":  analysisResult,
		"tx_hash": txHash,
	}
	explainJSON, _ := json.Marshal(fulfilledExplain)

	// Update proposal as fulfilled
	if err := t.proposalService.UpdateProposalFulfilled(
		proposal.ProposalID,
		proposal.DaoCode,
		string(explainJSON),
	); err != nil {
		return fmt.Errorf("vote cast but failed to update proposal status: %w", err)
	}

	return nil
}

// queryAllVotes queries all votes for a proposal with pagination
func (t *ProposalFulfillTask) queryAllVotes(ctx context.Context, indexer *internal.DegovIndexer, proposalID string) ([]internal.VoteCast, error) {
	var allVotes []internal.VoteCast
	offset := 0

	for {
		votes, err := indexer.QueryVotesOffset(ctx, offset, proposalID)
		if err != nil {
			return nil, err
		}

		if len(votes) == 0 {
			break
		}

		allVotes = append(allVotes, votes...)
		offset += len(votes)

		// Safety limit to prevent infinite loops
		if offset > 10000 {
			slog.Warn("[proposal-fulfill] Reached vote query limit",
				"proposal_id", proposalID,
				"total_votes", len(allVotes))
			break
		}
	}

	return allVotes, nil
}

// parseInt64 parses a string to int64
func parseInt64(s string) (int64, error) {
	var result int64
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
