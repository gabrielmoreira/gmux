# gmux monorepo (scaffold)

Clean-slate rewrite of the gmux ecosystem.

## Goals

- Clean boundaries between UI/backend, node agent, and wrapper runtime.
- Shared protocol contracts and schema-first evolution.
- Fast local developer experience with moon orchestration.
- Incremental migration from `agent-cockpit` without blindly copying old decisions.

## Workspace layout

- `apps/gmux-web` — Preact frontend
- `apps/gmux-api` — TypeScript backend (tRPC for UI, REST client to gmuxd)
- `packages/protocol` — shared contracts and schemas
- `services/gmuxd` — native node daemon (Go scaffold)
- `cli/gmux-run` — native launcher/wrapper (Go scaffold)
- `docs/` — ADRs, protocol specs, migration plans

## Tooling

- Monorepo orchestration: moon (`@moonrepo/cli`)
- JS/TS package manager: pnpm
- Native services: Go modules (`services/gmuxd`, `cli/gmux-run`)
- VCS: jj (Git backend)

## Quick commands

```bash
pnpm install
pnpm graph
moon projects
moon tasks
```

See `docs/plans/migration-plan-v1.md` for phased migration from current code.
