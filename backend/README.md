# DeGov Apps Backend

The backend no longer includes `degov-agent` automatic voting or proposal vote backfill.

`proposalSummary` is still available.

Legacy GraphQL queries related to removed agent-voting features are kept as deprecated compatibility endpoints and now return empty values so older frontends do not fail during rollout.
