# DeGov Apps Backend

The backend no longer includes `degov-agent` automatic voting or proposal vote backfill.

`proposalSummary` is still available.

Legacy GraphQL queries related to removed agent-voting features are kept as deprecated compatibility endpoints and now return empty values so older frontends do not fail during rollout.

## Task runner

Use the backend `justfile` for local orchestration.

- `just --list` shows available backend recipes
- `just run` starts the backend directly
- `just serve` builds `bin/degov-server` and serves it
- `just test` runs the backend test suite
