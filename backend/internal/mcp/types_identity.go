package mcp

type addressIdentityOutput struct {
	Address string  `json:"address"`
	ENSName *string `json:"ensName,omitempty"`
}
