---
schema: conductor/repository-workflow-policy/1
execution:
  canonicalize_commands: []
  verify_commands:
    - cd web && pnpm build
    - cd backend && just build
  max_attempts: 3
  retry_backoff_seconds: 60
  command_timeout_seconds: 1800
context:
  read_first:
    - README.md
    - web/README.md
    - backend/README.md
landing:
  default_merge_method: squash
  allowed_merge_methods:
    - merge
    - squash
---

Use this repository policy as the working contract for Conductor-owned lanes in DeGov Square.

DeGov Square is the DAO hub for organizations using the DeGov governance toolkit. The repository
contains a Next.js web app and a Go backend service. Keep each change scoped to the leased issue
and preserve the existing frontend/backend boundaries.

## Repository layout

- `web`: Next.js application, DAO onboarding/configuration UI, wallet integration, generated
  runtime assets, and frontend package metadata.
- `backend`: Go GraphQL/API service, persistence models, migrations, background tasks, DAO sync,
  notification logic, and backend-local task runner.
- `backend/migrations`: database schema migrations. Treat migration changes as stateful and test
  them against an intentional local or test database.
- `web/public/config.yml`, `web/.env.example`, `backend/.env.example`, and deployment workflows:
  runtime and environment configuration.

## Command policy

Use scoped commands from the package or service directory that owns the change.

The default non-mutating verification gate is:

```sh
cd web && pnpm build
cd backend && just build
```

Run additional scoped commands only when the touched area requires them:

- Frontend dependency changes: `cd web && pnpm install --frozen-lockfile` before build.
- Frontend runtime, route, component, config, or style changes: `cd web && pnpm build`.
- Backend service, GraphQL resolver, model, task, or helper changes: `cd backend && just test`
  when the test environment is available; otherwise run `cd backend && just build` and record
  the limitation.
- Backend GraphQL schema or generated code changes: `cd backend && just generate`, then
  `cd backend && just build`, and include generated files in the change.
- Backend dependency changes: `cd backend && just deps`, then `cd backend && just build`.
- Migration changes: validate against an intentional local/test database only. Do not run
  production migrations from Conductor.

Do not run deployment, production database, chain-writing, or notification-sending commands unless
the issue explicitly requests that exact operation and the required environment is intentionally
provided.

## Execution rules

Read `README.md`, `web/README.md`, and `backend/README.md` before changing code. For backend work,
also inspect the relevant service, task, model, migration, or GraphQL schema files before editing.

Keep secrets out of durable surfaces: issue comments, PR bodies, commit messages, logs, test
fixtures, generated config, and documentation. Use environment variables and example values only.

Use Conductor tracker tools for attempt results, terminal records, review handoff, repair
completion, and closeout. Do not hand-write lifecycle state into commit messages or issue
comments when a structured tracker record exists.
