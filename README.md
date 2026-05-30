# CLIProxyAPI Personal Fork

This repository is a personal fork of [`router-for-me/CLIProxyAPI`](https://github.com/router-for-me/CLIProxyAPI). It exists to keep a reproducible public build of the changes I use on top of upstream CLIProxyAPI.

This fork is published for personal use only. It is provided as-is, with no warranty, support commitment, or responsibility assumed by the maintainer for any usage, deployment, account, billing, data, compliance, or service-availability consequences.

## Why This Fork Exists

The main reason for this fork is to make Claude Code and other CLI AI clients work more predictably when routed through third-party Anthropic-compatible, OpenAI-compatible, Gemini-compatible, or mixed provider backends.

In practice, these clients often rely on subtle protocol behavior: mid-conversation system messages, tool history, thinking/reasoning blocks, response signatures, model suffixes, retry behavior, and request logging. Small differences in how a proxy translates or normalizes those fields can break provider compatibility, invalidate prefix cache assumptions, or make debugging difficult.

This fork keeps the upstream architecture where possible, but carries practical changes that are useful for my deployment and easier to audit from source.

## What Is Different From Upstream

Recent changes in this fork focus on:

- Preserving Claude mid-conversation `role: "system"` messages when the request enables Anthropic's `mid-conversation-system-*` beta.
- Deduplicating repeated Claude Code task reminder system messages before requests are forwarded upstream.
- Avoiding unnecessary movement of Claude Code runtime system reminders into the top-level Anthropic `system` field when the upstream request supports mid-conversation system messages.
- Preserving content forwarding across provider bridges.
- Improving cross-provider switching behavior for thinking blocks, reasoning history, tool history, response models, and normalized messages.
- Adding or carrying compatibility fixes around response WebSocket input item deduplication, `count_tokens` auth state handling, model suffix stripping, and provider signature validation.
- Keeping request logging and local debugging support available for verifying the exact upstream payload sent by the proxy.
- Adding a small checkpoint helper script for local development snapshots.

This is not meant to be a general-purpose replacement for upstream CLIProxyAPI. It is a personal working fork with targeted behavior changes.

## Build From Source

Requirements:

- Go 1.26 or later
- Git

Build the server binary:

```bash
git clone https://github.com/Nouvelle-Lune/CLIProxyAPI.git
cd CLIProxyAPI
go mod download
go build -o cli-proxy-api ./cmd/server
```

Create a local config and run the server:

```bash
cp config.example.yaml config.yaml
./cli-proxy-api --config config.yaml
```

For development, you can also run without creating a binary:

```bash
go run ./cmd/server --config config.yaml
```

## Local Files And Secrets

Do not commit local runtime data or secrets. This fork intentionally ignores local deployment files such as:

- `config.yaml`
- `cliproxyapi.conf`
- `.env`
- `auths/`
- `logs/`
- local build binaries
- local rebuild/restart scripts
- temporary working files

Before publishing or pushing any changes, check the repository for API keys, OAuth tokens, personal config files, request logs, and local paths.

## Upstream

Original project:

```text
https://github.com/router-for-me/CLIProxyAPI
```

This fork should be read as a personal downstream variant, not as official documentation for the upstream project.
