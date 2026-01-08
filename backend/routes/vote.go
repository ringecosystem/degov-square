package routes

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ringecosystem/degov-square/services"
)

type VoteRoute struct {
	proposalService  *services.ProposalService
	daoService       *services.DaoService
	daoConfigService *services.DaoConfigService
}

// NewVoteRoute creates a new vote route handler
func NewVoteRoute() *VoteRoute {
	return &VoteRoute{
		proposalService:  services.NewProposalService(),
		daoService:       services.NewDaoService(),
		daoConfigService: services.NewDaoConfigService(),
	}
}

// VoteResponse represents the API response for vote endpoint
type VoteResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data,omitempty"`
	Msg  string      `json:"msg,omitempty"`
}

// VoteData represents the vote data in the response
type VoteData struct {
	ID               string                 `json:"id"`
	DaoCode          string                 `json:"daocode"`
	ProposalID       string                 `json:"proposal_id"`
	ChainID          int                    `json:"chain_id"`
	Status           string                 `json:"status"`
	Errored          int                    `json:"errored"`
	Fulfilled        int                    `json:"fulfilled"`
	FulfilledExplain map[string]interface{} `json:"fulfilled_explain"`
	CTime            string                 `json:"ctime"`
	UTime            string                 `json:"utime,omitempty"`
	DAO              interface{}            `json:"dao,omitempty"`
}

// VoteHandler handles the /degov/vote/:chain/:id endpoint
// GET /degov/vote/{chain}/{id}?format=json&fulfilled=1
func (v *VoteRoute) VoteHandler(w http.ResponseWriter, r *http.Request) {
	// Get path parameters
	chainStr := r.PathValue("chain")
	proposalID := r.PathValue("id")

	if chainStr == "" || proposalID == "" {
		v.sendError(w, http.StatusBadRequest, "chain and id are required")
		return
	}

	chainID, err := strconv.Atoi(chainStr)
	if err != nil {
		v.sendError(w, http.StatusBadRequest, "invalid chain id")
		return
	}

	// Get query parameters
	queryParams := r.URL.Query()
	format := queryParams.Get("format")
	if format == "" {
		format = "json"
	}

	// Get fulfilled filter (optional)
	var fulfilledFilter *int
	if fulfilledStr := queryParams.Get("fulfilled"); fulfilledStr != "" {
		fulfilled, err := strconv.Atoi(fulfilledStr)
		if err == nil {
			fulfilledFilter = &fulfilled
		}
	}

	// Find proposal by chain and proposal ID
	proposal, err := v.proposalService.FindByChainAndProposalID(chainID, proposalID, fulfilledFilter)
	if err != nil || proposal == nil {
		v.sendError(w, http.StatusNotFound, "proposal not found")
		return
	}

	// Get DAO info
	dao, _ := v.daoService.GetByCode(proposal.DaoCode)

	// Parse fulfilled_explain JSON
	var fulfilledExplain map[string]interface{}
	if proposal.FulfilledExplain != nil && *proposal.FulfilledExplain != "" {
		json.Unmarshal([]byte(*proposal.FulfilledExplain), &fulfilledExplain)
	}
	if fulfilledExplain == nil {
		fulfilledExplain = make(map[string]interface{})
	}

	// Build response
	var utime string
	if proposal.UTime != nil {
		utime = proposal.UTime.Format("2006-01-02T15:04:05Z")
	}

	data := VoteData{
		ID:               proposal.ID,
		DaoCode:          proposal.DaoCode,
		ProposalID:       proposal.ProposalID,
		ChainID:          proposal.ChainId,
		Status:           string(proposal.State),
		Errored:          proposal.FulfillErrored,
		Fulfilled:        proposal.Fulfilled,
		FulfilledExplain: fulfilledExplain,
		CTime:            proposal.CTime.Format("2006-01-02T15:04:05Z"),
		UTime:            utime,
		DAO:              dao,
	}

	v.sendSuccess(w, data)
}

func (v *VoteRoute) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(VoteResponse{
		Code: 0,
		Data: data,
	})
}

func (v *VoteRoute) sendError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(VoteResponse{
		Code: -1,
		Msg:  msg,
	})
}
