# DeGov Square

DeGov Square is a hub for DAOs that use the [degov](https://github.com/ringecosystem/degov) toolkit for governance. The setup files for these DAOs are available in the [degov-registry](https://github.com/ringecosystem/degov-registry).

If your DAO uses DeGov for governance, feel free to submit a pull request to add it to the registry.

## MCP usage

DeGov Square can expose its governance data through a Model Context Protocol (MCP) server. The MCP server is implemented by the backend as a Streamable HTTP endpoint, so agents connect to an HTTP URL such as `http://localhost:8080/mcp` or `https://your-domain.example/mcp`. It is not a stdio MCP server.

The current MCP tools are read-only and cover DAO discovery, public DAO config, tracked proposals, proposal state, proposal summaries, contributors, and proposal votes:

- `ping`
- `list_daos`
- `get_dao`
- `get_dao_config`
- `list_proposals`
- `get_proposal`
- `get_proposal_state`
- `summarize_proposal`
- `get_contributor`
- `list_contributors`
- `list_proposal_votes`

These tools read from DeGov Square's database and configured DAO indexers. They do not create proposals, cast votes, execute transactions, or write governance state.

### Backend configuration

The MCP endpoint is disabled by default. Enable it on the backend with these environment variables:

```sh
MCP_ENABLED=true
MCP_PATH=/mcp
MCP_AUTH_MODE=bearer
MCP_BEARER_TOKEN=replace-with-a-long-random-token
```

`MCP_AUTH_MODE` supports:

- `bearer`: require `Authorization: Bearer <token>`. This is the default and should be used for shared or public deployments.
- `none`: no MCP authentication. Use only for trusted local development.
- `oauth`: require an OAuth bearer token validated with the configured issuer, JWKS URL, audience, and scopes.
- `bearer,oauth`: accept OAuth tokens and, when `MCP_BEARER_TOKEN` is set, also accept the static bearer token.

OAuth deployments can use these variables when needed:

```sh
MCP_OAUTH_RESOURCE=https://your-domain.example/mcp
MCP_OAUTH_AUTHORIZATION_SERVERS=https://issuer.example
MCP_OAUTH_ISSUER=https://issuer.example
MCP_OAUTH_JWKS_URL=https://issuer.example/.well-known/jwks.json
MCP_OAUTH_AUDIENCE=degov-square-mcp
MCP_OAUTH_SCOPES_SUPPORTED=degov.mcp.read
MCP_OAUTH_REQUIRED_SCOPES=degov.mcp.read
MCP_OAUTH_ALLOW_STATIC_BEARER=false
```

When OAuth mode is enabled, the backend also serves protected resource metadata at `/.well-known/oauth-protected-resource` and `/.well-known/oauth-protected-resource/mcp`.

Proposal summary generation is disabled by default. `summarize_proposal` can return cached summaries without generation. To let the MCP server generate a missing or refreshed summary, configure:

```sh
MCP_PROPOSAL_SUMMARY_GENERATE_ENABLED=true
MCP_PROPOSAL_SUMMARY_TIMEOUT=30s
OPENROUTER_API_KEY=...
```

### Run the backend locally

From the backend directory, start Postgres and run the server:

```sh
cd backend
DB_NAME=degov_square DB_PASSWORD=postgres DB_PORT=5432 just docker-up

export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=degov_square
export DB_SSLMODE=disable
export MCP_ENABLED=true
export MCP_AUTH_MODE=bearer
export MCP_BEARER_TOKEN=dev-mcp-token

just run
```

The backend loads `.env` from the current working directory, so you can put these values in `backend/.env` instead of exporting them in your shell.

For local-only testing without a token:

```sh
MCP_ENABLED=true MCP_AUTH_MODE=none just run
```

Do not use `MCP_AUTH_MODE=none` on a network-accessible deployment.

### Connect from agents and MCP clients

Use the backend MCP URL and the same bearer token configured in `MCP_BEARER_TOKEN`.

#### Claude Code

Claude Code supports remote HTTP MCP servers:

```sh
claude mcp add --transport http degov-square http://localhost:8080/mcp \
  --header "Authorization: Bearer dev-mcp-token"
```

Then restart or reload Claude Code and ask it to use the `degov-square` MCP server, for example: "Use `degov-square` to list DAOs."

#### Cursor

Create `.cursor/mcp.json` in a project, or `~/.cursor/mcp.json` for a global configuration:

```json
{
  "mcpServers": {
    "degov-square": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer dev-mcp-token"
      }
    }
  }
}
```

Replace `dev-mcp-token` with the real token for your backend. Do not commit a shared `.cursor/mcp.json` that contains a production token. Cursor shows MCP tools under its available tools list and asks for tool approval according to your Cursor run mode.

#### Codex

Codex uses `config.toml` for MCP servers. Add this to `~/.codex/config.toml`:

```toml
[mcp_servers.degov_square]
url = "http://localhost:8080/mcp"
bearer_token_env_var = "DEGOV_SQUARE_MCP_TOKEN"
tool_timeout_sec = 60
```

Before starting Codex, set:

```sh
export DEGOV_SQUARE_MCP_TOKEN=dev-mcp-token
```

Inside Codex, use `/mcp` to confirm the server is connected.

#### Other Streamable HTTP MCP clients

Use this connection information:

- URL: `http://localhost:8080/mcp` for local development, or your deployed `MCP_PATH`.
- Transport: Streamable HTTP.
- Authentication header for bearer mode: `Authorization: Bearer <MCP_BEARER_TOKEN>`.
- OAuth mode: use the server's protected resource metadata and request the configured `MCP_OAUTH_REQUIRED_SCOPES`.

If a client only supports stdio MCP servers, it cannot connect directly to DeGov Square's backend. Use a client that supports Streamable HTTP, or put an MCP-compatible HTTP-to-stdio bridge in front of the backend.

### Verify the connection

Start with `ping`; a healthy MCP connection returns `status: "ok"` with service `degov-square`.

Then try a read-only governance query:

```json
{
  "tool": "list_daos",
  "arguments": {
    "limit": 5
  }
}
```

For DAO-specific tools, use a real DAO code returned by `list_daos`, for example:

```json
{
  "tool": "get_dao_config",
  "arguments": {
    "daoCode": "ring-dao",
    "format": "json"
  }
}
```

If the MCP client reports an authentication failure, check that `MCP_AUTH_MODE` matches the client configuration and that the `Authorization` header is present. If DAO, proposal, contributor, or vote tools return missing data, confirm the database has been migrated and populated, and that the DAO's indexer endpoint in the registry config is reachable.

### Notes and limitations

- Tool results are intentionally bounded. For example, `list_daos` is capped at 100 rows, and proposal, contributor, and vote list tools are capped at 50 rows.
- `get_dao_config` supports `format: "json"` and `format: "yaml"`.
- Proposal states use the backend's tracked governance states such as `PENDING`, `ACTIVE`, `SUCCEEDED`, `EXECUTED`, and related enum values.
- ENS names are resolved on a best-effort basis when contributor, voter, or proposer identity data is returned.
