package mcp

type resolveENSInput struct {
	Name    string `json:"name,omitempty" jsonschema:"ENS name to resolve."`
	Address string `json:"address,omitempty" jsonschema:"EVM address to reverse-resolve."`
}

type resolveENSRecordsInput struct {
	Name string `json:"name" jsonschema:"ENS name to inspect."`
}

type resolveENSOutput struct {
	Name    *string `json:"name,omitempty"`
	Address *string `json:"address,omitempty"`
}

type resolveENSRecordsOutput struct {
	Name        string            `json:"name"`
	Address     *string           `json:"address,omitempty"`
	Contenthash *string           `json:"contenthash,omitempty"`
	Text        map[string]string `json:"text,omitempty"`
}
