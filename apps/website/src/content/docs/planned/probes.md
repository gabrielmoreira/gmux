---
title: Probes
description: Planned folder-level metadata and project context.
---

> This feature is not yet implemented.

Probes would add project-level context to folder groups in the sidebar — things like git branch, CI status, or dependency health. They're distinct from adapters, which operate per-session.

Open questions:

- Where probes run (gmuxd, shell scripts, or plugins)
- What the plugin/script contract looks like
- How probe results affect sorting, grouping, and folder display
