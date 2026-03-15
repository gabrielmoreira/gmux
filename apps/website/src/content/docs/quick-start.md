---
title: Quick Start
description: Current local-development path for running gmux.
---

> TODO: replace this page with end-user install and run instructions once packaging and distribution are settled.

This page documents the workflow that is accurate in the repository today: starting gmux from a local checkout.

## From a local checkout

Install dependencies:

```bash
pnpm install
```

Start the development stack:

```bash
pnpm dev
```

This starts the relevant services in watch mode, including the browser UI and backend components used during development.

Open the UI:

```bash
open http://localhost:5173
```

## What to expect

In local development, the main dev services are:

- `gmux-web` on `http://localhost:5173`
- `gmux-api` on `http://localhost:8787`
- `gmuxd` on `http://localhost:8790`

The root dev script also starts the supporting watchers needed for a working development environment.

## Launching sessions

Once the stack is running, sessions launched through `gmuxr` should appear in the UI.

Examples:

```bash
gmuxr pi
gmuxr -- make build
gmuxr -- pytest --watch
```

## TODO

- Document the production / installed workflow separately from local development.
- Explain how the browser UI is expected to be served outside the monorepo dev setup.
- Add a short verification checklist for first run: which ports should respond, what a healthy sidebar looks like, and how to attach to a session.

## Next steps

- [Introduction](/introduction)
- [Architecture](/architecture)
- [Adapters](/adapters)
