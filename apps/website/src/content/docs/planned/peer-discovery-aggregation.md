---
title: Peer Discovery & Aggregation
description: See sessions from every machine, container, and VM in one dashboard.
---

> This feature is not yet implemented.

Today each gmuxd instance is an island. You open `gmux-desktop.tailnet.ts.net` to see your desktop sessions and `gmux-server.tailnet.ts.net` to see your server sessions. If you run coding agents on three machines, you juggle three tabs.

Cross-instance lets any gmuxd show sessions from other gmuxd instances alongside its own. One URL, every machine.

## The model

Every gmuxd is a **peer**. Peers connect to each other over the same HTTP/SSE/WS protocol that the browser already uses. When peer A connects to peer B, it subscribes to B's `/v1/events` SSE stream and merges B's sessions into its own store. When the browser asks A for sessions, it gets a unified list. When the browser opens a terminal on one of B's sessions, A proxies the WebSocket through to B.

There is no primary. Any peer can connect to any other peer. In practice, you'll probably pick one machine as your "home" dashboard, but nothing in the architecture requires it.

```
browser ──→ gmux-laptop (peer)
               ├── local sessions
               ├── ← gmux-desktop (peer, via tailscale)
               │      └── desktop sessions
               └── ← gmux-server (peer, via tailscale)
                      ├── server sessions
                      └── container: project-a sessions
```

## Discovery

### Tailscale auto-discovery

gmuxd instances already register as tailscale devices. Tailscale's local API (`/api/v0/status`) lists all nodes on the tailnet. gmuxd can query this periodically and look for peers:

1. List all online nodes on the tailnet.
2. Filter to nodes tagged with `tag:gmux` (via tailscale ACL tags) or matching a configurable hostname pattern.
3. For each candidate, probe `https://<hostname>/v1/sessions` to confirm it's a gmuxd.
4. Subscribe to its `/v1/events` stream.

This gives you zero-config discovery on a tailnet: install gmux on two machines, they find each other.

### Manual peers

For cases where auto-discovery doesn't apply (containers on a Docker network, VMs on a private subnet, instances behind firewalls), peers can be configured explicitly:

```toml
[[peers]]
name = "server"
url = "https://gmux-server.tailnet.ts.net"

[[peers]]
name = "project-a"
url = "http://project-a:8790"
```

Manual peers are tried on startup and reconnected on failure. Auto-discovered and manual peers use the same protocol.

### Container discovery

Containers are a special case of manual peers. A host gmuxd could auto-discover containers by scanning Docker for containers with a `gmux.peer=true` label and connecting to their internal IP on port 8790. This is container-specific glue on top of the general peer protocol.

See the [Devcontainers](#devcontainers) section for the full picture.

## Data model

### Namespaced sessions

Each peer's sessions are prefixed with the peer name to avoid ID collisions:

```
local session:  sess-abc123
remote session: server/sess-def456
```

The store distinguishes local sessions (discovered via Unix sockets as today) from remote sessions (received via peer SSE). Remote sessions are read-through: the hub caches them for rendering but delegates actions (kill, resume, launch) to the owning peer.

### Canonical project URI

Sessions gain a `project_uri` field: a normalized identifier derived from the VCS remote URL.

```
git@github.com:gmuxapp/gmux.git  →  github.com/gmuxapp/gmux
https://github.com/gmuxapp/gmux  →  github.com/gmuxapp/gmux
```

The runner detects this at session startup (it already walks up from cwd to find `workspace_root`; reading `git remote get-url origin` or `jj git remote list` is one more step). The field is included in the `/meta` response alongside `workspace_root`.

The UI uses `project_uri` to group sessions across peers. Two sessions on different machines with the same `project_uri` appear under one project heading:

```
gmuxapp/gmux
  Fix auth bug          laptop     pi
  Run tests             server     shell
  Refactor adapter      server     pi (container: project-a)
```

The local filesystem path (`cwd`, `workspace_root`) is still shown in session details. The canonical URI is for grouping only.

When `project_uri` is empty (no VCS remote, or a local-only repo), sessions fall back to grouping by `workspace_root` or `cwd` as they do today, scoped to their peer.

### Peer metadata

Each peer advertises metadata alongside its sessions:

```json
{
  "name": "gmux-server",
  "version": "0.9.0",
  "os": "linux",
  "hostname": "unraid"
}
```

The UI uses this for display (icons, labels) and compatibility checks. A peer running an incompatible protocol version is flagged but not rejected.

## Protocol

### Subscribing

The hub connects to `GET /v1/events` on each peer, the same SSE endpoint the browser uses. Session upsert/remove events are prefixed with the peer name and merged into the hub's store.

Authentication uses the same mechanism as browser connections: tailscale identity for tailnet peers, bearer token for network-listener peers. No new auth scheme.

### Proxying

When the browser opens a terminal on a remote session (`WS /ws/server/sess-def456`), the hub:

1. Strips the peer prefix to get the session ID (`sess-def456`).
2. Opens a WebSocket to the owning peer (`WS /ws/sess-def456` on the server).
3. Relays frames bidirectionally.

This is similar to what gmuxd already does between the browser and local runner sockets; the proxy just has one more hop.

### Launching

`POST /v1/launch` accepts an optional `peer` field. When set, the hub forwards the request to that peer's `/v1/launch` endpoint. The peer launches the session locally and the hub discovers it via the SSE stream.

```json
{
  "launcher": "pi",
  "cwd": "/workspace/gmux",
  "peer": "server"
}
```

When `peer` is omitted, the session launches locally (current behavior).

### Fault tolerance

Peer connections are resilient:

- If a peer goes offline, its sessions are marked as disconnected (grey) in the UI. They remain visible so you know what was running.
- When the peer comes back, the hub re-subscribes and sessions go live again.
- The hub never blocks on a slow or dead peer. Each peer subscription is independent.

## UI changes

### Sidebar

The sidebar shifts from "folders on this machine" to "projects across all peers." Grouping priority:

1. **By `project_uri`** when available (cross-machine project grouping).
2. **By `workspace_root`** when sessions share a VCS root on the same peer (the existing workspace grouping from [Folder Management](/planned/folder-management)).
3. **By `cwd`** as a final fallback.

Each session shows a subtle peer indicator (hostname or icon) so you can tell where it's running. Sessions on the local peer have no indicator.

### Peer status

A footer or header element shows connected peers with their status (online, reconnecting, offline). Clicking a peer could filter the sidebar to only show that peer's sessions.

### Launch target

The launch modal gains a peer selector. When you have peers configured, you choose where to launch. The default is the local machine. For projects that have an associated devcontainer, the peer selector could auto-suggest the right container.

## Devcontainers

Cross-instance provides the connection layer. Devcontainer support builds on top.

### Architecture

Each devcontainer runs its own gmuxd, listening on the Docker network. The host gmuxd connects to it as a peer. No shared volumes, no socket leakage between containers.

```
host gmuxd
  ├── local sessions
  ├── ← container-a gmuxd (peer, Docker network)
  │      └── container-a sessions
  └── ← container-b gmuxd (peer, Docker network)
         └── container-b sessions
```

The container's gmuxd does not need tailscale. It uses the network listener (`network.listen = "0.0.0.0"`) with bearer-token auth on the Docker bridge. The host gmuxd is the only client.

### Lifecycle

When a project folder has a `.devcontainer/devcontainer.json`, gmuxd manages the container lifecycle:

1. **Start**: `devcontainer up --workspace-folder <path>` builds and starts the container if needed.
2. **Connect**: gmuxd connects to the container's gmuxd as a peer.
3. **Launch**: sessions in this project are launched inside the container via the peer's `/v1/launch`.
4. **Stop**: containers can be stopped from the UI or left running (user preference).

The `devcontainer` CLI handles all the complexity of building images, installing features, applying dotfiles. gmuxd just calls it.

### gmux as a devcontainer Feature

A devcontainer Feature (`ghcr.io/gmuxapp/features/gmux`) installs gmux and gmuxd into any devcontainer. Users add it to their `devcontainer.json`:

```json
{
  "image": "mcr.microsoft.com/devcontainers/base:debian",
  "features": {
    "ghcr.io/gmuxapp/features/gmux": {}
  }
}
```

The feature:
- Installs `gmux` and `gmuxd` binaries.
- Configures gmuxd to listen on `0.0.0.0:8790` with a generated bearer token.
- Sets up gmuxd as the container entrypoint (or an init process alongside the user's entrypoint).
- Writes the bearer token to a well-known path so the host gmuxd can read it.

### Dotfiles

Devcontainers have native dotfiles support. Users configure their dotfiles repo in `devcontainer.json` or their editor settings, and the devcontainer CLI clones and installs them on container creation. This is orthogonal to gmux; gmux doesn't need to know about dotfiles.

## Incremental delivery

### Step 1: Canonical project URI

Add `project_uri` to the runner's session metadata. Detected from VCS remote at session startup. Included in `/meta` response, stored in `store.Session`, broadcast via SSE. The frontend uses it for grouping within a single gmuxd instance (replaces path-based grouping for repos with remotes).

Small, self-contained. Useful today for workspace grouping even without cross-instance.

### Step 2: Peer protocol

gmuxd can connect to other gmuxd instances as peers. Manual `[[peers]]` config. Session namespacing, SSE subscription, WebSocket proxying, launch forwarding. No auto-discovery yet.

This unlocks the "one dashboard, every machine" use case for users with explicit config.

### Step 3: Tailscale auto-discovery

gmuxd queries the tailnet for other gmux instances. Zero-config for the common case. Builds on the peer protocol from step 2.

### Step 4: Devcontainer integration

gmuxd detects `.devcontainer/devcontainer.json` in project folders. Manages container lifecycle via the `devcontainer` CLI. Connects to container gmuxd as a peer. gmux devcontainer Feature for easy installation.

Each step is independently useful and shippable.
