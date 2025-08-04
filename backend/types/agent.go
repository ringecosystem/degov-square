package types

type AgentDaoConfig struct {
	Name               string   `json:"name"`
	Code               string   `json:"code"`
	XProfile           string   `json:"xprofile"`
	Carry              []string `json:"carry"`
	LastProcessedBlock int64    `json:"last_processed_block"`
}
