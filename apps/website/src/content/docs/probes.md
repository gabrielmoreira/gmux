---
title: Probes
description: How gmux enriches folder headings with project context.
---

Probes are directory-level intelligence. They observe working directories and report context that enriches the sidebar's folder headings.

## What probes provide

When sessions share a working directory, they're grouped into a **folder**. Probes add metadata to the folder heading:

```
▼ myapp                        ● 2
  main · 3 changed · PR #42
```

Here, the **git probe** reports `main · 3 changed` and the **GitHub PR probe** reports `PR #42`.

## Built-in probes

| Probe | Reports |
|-------|---------|
| **git** | Branch name, dirty file count |
| **github-pr** | Open PR number, status |

## Script probes

Drop a bash script in `~/.config/gmux/probes/` and it becomes a probe. gmuxd runs it against each matching directory and expects JSON output.

### Example: Node.js version probe

```bash
#!/bin/bash
# ~/.config/gmux/probes/node-version.sh

if [ -f "$1/package.json" ]; then
  version=$(node -v 2>/dev/null)
  echo "{\"label\": \"$version\"}"
fi
```

This adds the Node.js version to any folder containing a `package.json`.

### Script probe contract

- **Input**: The working directory path is passed as `$1`
- **Output**: A JSON object on stdout with at least a `label` field
- **No output**: If the probe doesn't apply, output nothing (exit silently)
- **Timeout**: Probes are killed after 5 seconds
