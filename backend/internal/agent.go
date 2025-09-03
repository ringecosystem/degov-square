package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type VoteResponse struct {
	Code int      `json:"code"`
	Data VoteData `json:"data"`
}

type VoteData struct {
	ID                string                 `json:"id"`
	DaoCode           string                 `json:"daocode"`
	ProposalID        string                 `json:"proposal_id"`
	ChainID           int                    `json:"chain_id"`
	Status            string                 `json:"status"`
	Errored           int                    `json:"errored"`
	Fulfilled         int                    `json:"fulfilled"`
	Type              string                 `json:"type"`
	SyncStopTweet     int                    `json:"sync_stop_tweet"`
	SyncStopReply     int                    `json:"sync_stop_reply"`
	SyncNextTimeTweet string                 `json:"sync_next_time_tweet"`
	TimesProcessed    int                    `json:"times_processed"`
	FulfilledExplain  map[string]interface{} `json:"fulfilled_explain"`
	CTime             string                 `json:"ctime"`
	UTime             string                 `json:"utime"`
	DAO               map[string]interface{} `json:"dao"`
	TwitterUser       TwitterUser            `json:"twitter_user"`
}

type TwitterUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Verified bool   `json:"verified"`
}

type DegovAgent struct {
	BaseURL string
}

func NewDegovAgent() *DegovAgent {
	return &DegovAgent{
		BaseURL: "https://agent.degov.ai",
	}
}

func (agent *DegovAgent) QueryVote(chainId int, proposalId string) (*VoteData, error) {
	url := fmt.Sprintf("%s/degov/vote/%d/%s?format=json", agent.BaseURL, chainId, proposalId)
	slog.Debug("Querying vote API", "url", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("[degov-agent] failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[degov-agent] received non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[degov-agent] failed to read response body: %w", err)
	}

	var voteResponse VoteResponse
	if err := json.Unmarshal(body, &voteResponse); err != nil {
		return nil, fmt.Errorf("[degov-agent] failed to unmarshal JSON response: %w", err)
	}

	if voteResponse.Code != 0 {
		return nil, fmt.Errorf("[degov-agent] API returned an error code: %d", voteResponse.Code)
	}

	return &voteResponse.Data, nil
}
