# gmux monorepo (scaffold)

Clean-slate rewrite of the gmux ecosystem.

## Goals

- Clean boundaries between UI/backend, node agent, and wrapper runtime.
- Shared protocol contracts and schema-first evolution.
- Fast local developer experience with moon orchestration.
- Incremental migration from `agent-cockpit` without blindly copying old decisions.
- Web-first UX that works for local, server, and mobile access.

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

## Distribution direction (v1)

- Ship native binaries: `gmuxd` and `gmux-run`.
- TS apps are deployed runtime components, not npm products for end users.
- Web-first UI surface:
  - local mode: `gmuxd` serves UI
  - server mode: `gmux-api` serves/aggregates behind reverse proxy auth
  - mobile access uses the same web UI
- No Electron in v1; add a convenience `gmux open` app-mode launcher later.

## Quick commands

```bash
pnpm install
pnpm graph
pnpm moon projects
pnpm moon tasks
```

## jj + moon note

Moon calls `git` for VCS metadata. In a brand new `jj git init` repo, set Git HEAD once:

```bash
git symbolic-ref HEAD refs/heads/main
```

Without this, moon commands may fail with `HEAD not found below refs/heads`.

See `docs/plans/migration-plan-v1.md` for phased migration from current code.
