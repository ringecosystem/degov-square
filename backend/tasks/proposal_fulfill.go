package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/internal"
	"github.com/ringecosystem/degov-square/services"
	"github.com/ringecosystem/degov-square/types"
)

// FeatureFulfill is the feature name for AI-based voting
const FeatureFulfill = "fulfill"

// ProposalFulfillTask handles AI-based voting on proposals
type ProposalFulfillTask struct {
	daoService       *services.DaoService
	daoConfigService *services.DaoConfigService
	proposalService  *services.ProposalService
	openRouterClient *internal.OpenRouterClient
	fulfillAnalyzer  *services.FulfillAnalyzer
}

// NewProposalFulfillTask creates a new proposal fulfill task
func NewProposalFulfillTask() *ProposalFulfillTask {
	slog.Info("[proposal-fulfill] Task initialized, will query DAOs with 'fulfill' feature from database")

	openRouterClient := internal.NewOpenRouterClient()
	fulfillAnalyzer, err := services.NewFulfillAnalyzer(openRouterClient)
	if err != nil {
		slog.Error("[proposal-fulfill] Failed to create fulfill analyzer", "error", err)
		// Continue with nil analyzer - will fail gracefully when used
	}

	return &ProposalFulfillTask{
		daoService:       services.NewDaoService(),
		daoConfigService: services.NewDaoConfigService(),
		proposalService:  services.NewProposalService(),
		openRouterClient: openRouterClient,
		fulfillAnalyzer:  fulfillAnalyzer,
	}
}

// Name returns the task name
func (t *ProposalFulfillTask) Name() string {
	return "proposal-fulfill"
}

// Execute performs the proposal fulfill task
func (t *ProposalFulfillTask) Execute() error {
	slog.Info("[proposal-fulfill] ========== Execute() START ==========")
	err := t.fulfillProposals()
	slog.Info("[proposal-fulfill] ========== Execute() END ==========", "error", err)
	return err
}

// fulfillProposals processes unfulfilled proposals
func (t *ProposalFulfillTask) fulfillProposals() error {
	// Get DAOs with fulfill feature enabled from database
	supportedDAOs, err := t.daoService.ListDAOCodesWithFeature(FeatureFulfill)
	if err != nil {
		slog.Error("[proposal-fulfill] Failed to get DAOs with fulfill feature", "error", err)
		return err
	}

	slog.Info("[proposal-fulfill] fulfillProposals() called", "supportedDAOs", supportedDAOs)

	if len(supportedDAOs) == 0 {
		slog.Info("[proposal-fulfill] No DAOs have fulfill feature enabled, skipping task")
		return nil
	}

	// Get unfulfilled proposals (filtered by supported DAOs)
	proposals, err := t.proposalService.ListUnfulfilledProposals(supportedDAOs)
	if err != nil {
		slog.Error("[proposal-fulfill] Failed to list unfulfilled proposals", "error", err)
		return err
	}

	slog.Info("[proposal-fulfill] Query returned proposals", "count", len(proposals))

	// Log each proposal found
	for i, p := range proposals {
		slog.Info("[proposal-fulfill] Found proposal",
			"index", i,
			"proposal_id", p.ProposalID,
			"dao_code", p.DaoCode,
			"state", p.State,
			"fulfilled", p.Fulfilled,
			"fulfill_errored", p.FulfillErrored,
			"times_fulfill", p.TimesFulfill)
	}

	if len(proposals) == 0 {
		slog.Info("[proposal-fulfill] No unfulfilled proposals found, skipping task")
		return nil
	}

	slog.Info("[proposal-fulfill] Found unfulfilled proposals", "count", len(proposals))

	// Filter proposals that are ready (past midpoint)
	slog.Info("[proposal-fulfill] Calling filterReadyProposals...")
	readyProposals, err := t.filterReadyProposals(proposals)
	if err != nil {
		slog.Error("[proposal-fulfill] Failed to filter ready proposals", "error", err)
		return err
	}

	slog.Info("[proposal-fulfill] filterReadyProposals returned", "ready_count", len(readyProposals))

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

	slog.Info("[proposal-fulfill] filterReadyProposals() called", "proposal_count", len(proposals))

	// Get agent address for delegation check
	agentAddress := internal.GetAgentAddress()
	slog.Info("[proposal-fulfill] Agent address", "address", agentAddress)
	if agentAddress == "" {
		slog.Warn("[proposal-fulfill] Agent address not configured, skipping delegation check")
	}

	for _, proposal := range proposals {
		slog.Info("[proposal-fulfill] Processing proposal", "proposal_id", proposal.ProposalID, "dao_code", proposal.DaoCode)

		// Get DAO config
		daoConfig, err := t.daoConfigService.StandardConfig(proposal.DaoCode)
		if err != nil {
			slog.Warn("[proposal-fulfill] Failed to get DAO config",
				"dao_code", proposal.DaoCode,
				"error", err)
			continue
		}
		slog.Info("[proposal-fulfill] Got DAO config", "dao_code", proposal.DaoCode, "indexer", daoConfig.Indexer.Endpoint)

		// Create indexer to query proposal details
		indexer := internal.NewDegovIndexer(daoConfig.Indexer.Endpoint)

		// Check if agent has delegators (other than itself)
		if agentAddress != "" {
			ctx := context.Background()
			slog.Info("[proposal-fulfill] Checking delegators", "agent_address", agentAddress)
			hasDelegators, err := indexer.HasDelegatorsOtherThanSelf(ctx, agentAddress)
			slog.Info("[proposal-fulfill] Delegator check result", "has_delegators", hasDelegators, "error", err)
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
			} else {
				slog.Info("[proposal-fulfill] Agent has delegators, proceeding", "agent_address", agentAddress)
			}
		}

		// Query proposal to get vote window
		slog.Info("[proposal-fulfill] Querying proposal from indexer", "proposal_id", proposal.ProposalID)
		proposalData, err := indexer.InspectProposal(proposal.ProposalID)
		if err != nil {
			slog.Warn("[proposal-fulfill] Failed to query proposal",
				"proposal_id", proposal.ProposalID,
				"error", err)
			continue
		}
		slog.Info("[proposal-fulfill] Got proposal data",
			"proposal_id", proposal.ProposalID,
			"vote_start", proposalData.VoteStart,
			"vote_end", proposalData.VoteEnd,
			"clock_mode", proposalData.ClockMode)

		// Check if past midpoint but not expired
		if proposalData != nil && proposalData.VoteStart != "" && proposalData.VoteEnd != "" {
			voteStart, err1 := parseInt64(proposalData.VoteStart)
			voteEnd, err2 := parseInt64(proposalData.VoteEnd)

			if err1 == nil && err2 == nil && voteEnd > voteStart {
				midpoint := voteStart + (voteEnd-voteStart)/2
				nowSeconds := time.Now().Unix()
				voteEndSeconds := voteEnd

				slog.Info("[proposal-fulfill] Vote window (raw)",
					"vote_start", voteStart,
					"vote_end", voteEnd,
					"midpoint_raw", midpoint,
					"clock_mode", proposalData.ClockMode)

				// Check clock mode - if timestamp mode, values are in milliseconds
				if proposalData.ClockMode == "mode=timestamp" {
					midpoint = midpoint / 1000
					voteEndSeconds = voteEnd / 1000
					slog.Info("[proposal-fulfill] Converted to seconds (timestamp mode)",
						"midpoint", midpoint,
						"vote_end_seconds", voteEndSeconds)
				}

				slog.Info("[proposal-fulfill] Time comparison",
					"now_seconds", nowSeconds,
					"midpoint", midpoint,
					"vote_end_seconds", voteEndSeconds,
					"past_midpoint", nowSeconds >= midpoint,
					"before_end", nowSeconds <= voteEndSeconds)

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

		slog.Info("[proposal-fulfill] Proposal is READY for voting", "proposal_id", proposal.ProposalID, "dao_code", proposal.DaoCode)
		ready = append(ready, readyProposalItem{
			proposal:  proposal,
			daoConfig: daoConfig,
		})
	}

	slog.Info("[proposal-fulfill] filterReadyProposals completed", "ready_count", len(ready))
	return ready, nil
}

// fulfillProposal processes a single proposal with AI analysis and voting
func (t *ProposalFulfillTask) fulfillProposal(proposal *dbmodels.ProposalTracking, daoConfig *types.DaoConfig) error {
	// Check if analyzer is available
	if t.fulfillAnalyzer == nil {
		return fmt.Errorf("fulfill analyzer not initialized")
	}

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
	voteCastInfos := make([]services.VoteCastInfo, len(voteCasts))
	for i, vote := range voteCasts {
		timestamp := time.Now()
		if ts, err := parseInt64(vote.BlockTimestamp); err == nil {
			timestamp = time.Unix(ts/1000, (ts%1000)*1000000)
		}

		voteCastInfos[i] = services.VoteCastInfo{
			Support:        services.VoteSupportText(vote.Support),
			Reason:         vote.Reason,
			Weight:         vote.Weight,
			BlockTimestamp: timestamp,
		}
	}

	// Analyze votes with AI with exponential backoff and jitter
	const (
		maxRetries = 3
		baseDelay  = time.Second
	)
	var analysisResult *services.AnalysisResult
	for attempt := 0; attempt < maxRetries; attempt++ {
		analysisResult, err = t.fulfillAnalyzer.AnalyzeVotes(ctx, voteCastInfos)
		if err == nil {
			break
		}

		slog.Warn("[proposal-fulfill] AI analysis attempt failed, retrying",
			"proposal_id", proposal.ProposalID,
			"attempt", attempt+1,
			"error", err)

		if attempt < maxRetries-1 {
			// Exponential backoff: baseDelay * 2^attempt with jitter
			backoff := baseDelay * time.Duration(1<<attempt)
			// Add jitter: random value between 0 and 50% of backoff (in nanoseconds)
			// time.Duration is int64 in nanoseconds, so int64(backoff)/2 gives us 50%
			maxJitter := max(int64(backoff)/2, 1)
			jitter := time.Duration(rand.Int64N(maxJitter))
			delay := backoff + jitter
			slog.Debug("[proposal-fulfill] Retrying with backoff",
				"proposal_id", proposal.ProposalID,
				"delay", delay)
			time.Sleep(delay)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to generate AI response after %d attempts: %w", maxRetries, err)
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
	support := services.VoteSupportToNumber(analysisResult.FinalResult)
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
