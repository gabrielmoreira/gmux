---
title: Remote Access
description: Access gmux from your phone, tablet, or another machine over tailscale.
---

:::caution[Not a collaboration tool]
Remote access is designed for accessing **your own machine** from your other devices — your phone, tablet, or laptop. It is not intended for sharing terminal sessions with other people. Every connected user gets full, unrestricted shell access.
:::

By default, gmux only listens on localhost. To access it from another device — your phone on the couch, a laptop in another room, or a tablet on the go — you can enable the built-in tailscale listener.

## Why tailscale?

gmux gives you full terminal access to your machine. Exposing that to a network requires strong security guarantees:

- **Encrypted transport** — tailscale uses WireGuard, so all traffic is end-to-end encrypted. No one on the network can sniff your terminal sessions.
- **Cryptographic identity** — every connection is authenticated by tailscale's key exchange. You can't spoof a peer identity.
- **No ports to open** — tailscale punches through NATs. You don't need to open firewall ports or set up port forwarding.

gmux adds an **allow list** on top: only tailscale users you explicitly name can connect. Everyone else gets a 403.

## Setup

### 1. Install tailscale

If you haven't already, [install tailscale](https://tailscale.com/download) on both the machine running gmux and the device you want to connect from.

### 2. Configure gmux

Create or edit `~/.config/gmux/config.toml`:

```toml
[tailscale]
enabled = true
```

That's it. Your own tailscale account is automatically whitelisted — gmuxd detects the node owner at startup. The hostname defaults to `gmuxd`, making it available at `https://gmuxd.your-tailnet.ts.net`.

| Field | Description |
|---|---|
| `enabled` | Start the tailscale listener. Default `false`. |
| `hostname` | The machine name on your tailnet. Default `gmuxd`, giving you `gmuxd.your-tailnet.ts.net`. Change this if you run gmux on multiple machines. |

### 3. Restart gmuxd

```bash
# If gmuxd is running, kill it — gmuxr will auto-start it next time
pkill gmuxd
gmuxr pi  # or any command — gmuxd starts automatically
```

Look for the log line:

```
tsauth: node owner you@github auto-whitelisted
tsauth: listening on https://gmuxd (allowed: [you@github])
```

### 4. Connect

On your other device, open:

```
https://gmuxd.your-tailnet.ts.net
```

The connection is HTTPS with a valid certificate (issued automatically by tailscale via Let's Encrypt). No certificate warnings, no HTTP fallback.

## What's checked on every request

1. The connection must come through tailscale (the listener only accepts tailnet traffic).
2. gmuxd calls tailscale's `WhoIs` API to identify the connecting peer.
3. The peer's **login name** is checked against the `allow` list.
4. If the login name doesn't match, the request gets a `403 Forbidden` and the attempt is logged.

This check runs on every HTTP request and WebSocket upgrade — there are no session cookies or tokens that could be stolen.

## The localhost listener is unchanged

The tailscale listener is a second, independent listener. The localhost listener (`127.0.0.1:8790`) continues to work exactly as before, with no authentication. Local access is always available — you can't lock yourself out by misconfiguring the allow list.

## Adding other accounts

If you use multiple tailscale accounts (e.g. personal and work), add them to the allow list:

```toml
[tailscale]
enabled = true
allow = ["your-other-account@github"]
```

:::danger[Think twice before adding other people]
The `allow` list should ideally only contain accounts that belong to you. There are no permission levels — everyone on the list gets full shell access to your machine, can read all terminal output (including secrets), launch processes, and kill sessions. This is equivalent to giving someone your SSH key.
:::

## Troubleshooting

**"tsauth: could not determine node owner"** — gmuxd couldn't identify your tailscale account. Make sure tailscale is logged in (`tailscale status`) and try again.

**Can't reach the hostname** — Make sure both devices are on the same tailnet and that MagicDNS is enabled in your tailscale admin console.

**Certificate warning** — This shouldn't happen with tailscale's automatic HTTPS. If it does, check that HTTPS certificates are enabled in your [tailscale DNS settings](https://login.tailscale.com/admin/dns).
