# CLIProxyAPI Personal Fork

This repository is a personal fork of [`router-for-me/CLIProxyAPI`](https://github.com/router-for-me/CLIProxyAPI). It exists to keep a reproducible public build of the changes I use on top of upstream CLIProxyAPI.

This fork is published for personal use only. It is provided as-is, with no warranty, support commitment, or responsibility assumed by the maintainer for any usage, deployment, account, billing, data, compliance, or service-availability consequences.

## Why This Fork Exists

The main reason for this fork is to make Claude Code and other CLI AI clients work more predictably when routed through third-party Anthropic-compatible, OpenAI-compatible, Gemini-compatible, or mixed provider backends.

In practice, these clients often rely on subtle protocol behavior: mid-conversation system messages, tool history, thinking/reasoning blocks, response signatures, model suffixes, retry behavior, and request logging. Small differences in how a proxy translates or normalizes those fields can break provider compatibility, invalidate prefix cache assumptions, or make debugging difficult.

This fork keeps the upstream architecture where possible, but carries practical changes that are useful for my deployment and easier to audit from source.

## Structural Differences From Upstream

The fork README is a completely different document from the upstream README. Here is a summary of what each section looks like:

| Section | Upstream | Fork |
|---------|----------|------|
| **Title** | `CLI Proxy API` + multi-language links | `CLIProxyAPI Personal Fork` |
| **Sponsor** | 4 sponsors with detailed ad tables (PackyCode, AICodeMirror, BmoPlus, VisionCoder, APIKEY.FUN) | None |
| **Overview** | Full feature list (16 items) | None |
| **Getting Started** | Points to `help.router-for.me` docs | None |
| **Management API** | Points to external docs | None |
| **Usage Statistics** | CPA Usage Keeper, CPA-Manager-Plus intro | None |
| **Amp CLI Support** | Full Amp integration guide | None |
| **SDK Docs** | 4 SDK doc links | None |
| **Contributing** | Standard contribution guide | None |
| **Who is with us?** | 18 community projects based on CLIProxyAPI | None |
| **More choices** | 4 derivative projects | None |
| **License** | MIT License | None |
| **Why This Fork Exists** | None | Fork purpose and motivation |
| **What Is Different** | None | Detailed diff notes (fork-specific + upstream merge changes) |
| **Build From Source** | None | Full build steps |
| **Local Files And Secrets** | None | Security reminders |
| **Upstream** | None | Link to upstream repo |

In short: the fork README is a streamlined personal fork description. It removes all sponsor ads, feature lists, ecosystem project lists, and doc links from upstream, and replaces them with the fork's purpose, difference notes, and build guide. There is almost no overlapping content between the two.

## Fork-Specific Changes

Recent changes in this fork focus on:

- Preserving Claude mid-conversation `role: "system"` messages when the request enables Anthropic's `mid-conversation-system-*` beta.
- Deduplicating repeated Claude Code task reminder system messages before requests are forwarded upstream.
- Avoiding unnecessary movement of Claude Code runtime system reminders into the top-level Anthropic `system` field when the upstream request supports mid-conversation system messages.
- Preserving content forwarding across provider bridges.
- Improving cross-provider switching behavior for thinking blocks, reasoning history, tool history, response models, and normalized messages.
- Adding or carrying compatibility fixes around response WebSocket input item deduplication, `count_tokens` auth state handling, model suffix stripping, and provider signature validation.
- Keeping request logging and local debugging support available for verifying the exact upstream payload sent by the proxy.
- Adding a small checkpoint helper script for local development snapshots.

## Upstream Changes Carried

As of the latest merge from upstream (`7bb7116e`), this fork also carries the following upstream changes:

- **Claude translation protocol fixes**: OpenAI tool message merging for role alternation compliance, JSON key ordering preservation via gjson/sjson for KV cache stability, and Codex `web_search_call` streaming protocol alignment (`364aa229`).
- **Runtime auth removal**: `Manager.Remove` for deleting runtime auth and unscheduling associated tasks (`55440f0a`).
- **Cloudflare challenge retry**: Progressive rate-limiting cooldown for 403 Cloudflare challenges instead of hard credential suspension (`45f58d4f`, `77061aad`).
- **Codex fixes**: Avoid replaying orphan tool calls (`17af0891`), handle non-empty reasoning and content items (`0e3c809c`), cache reasoning replay items (`603a08fc`).
- **Gemini system role normalization**: Message-level system roles normalized for Gemini provider (`68282c4a`).
- **uTLS HTTP client enhancement**: Context-aware RoundTripper for protected hosts with Cloudflare bypass (`35ab084f`).
- **Home auth refresh fix**: Parse Home refresh auth envelopes so refreshed access tokens are used correctly (`c9dc6bd6`).
- **WebSocket input ID dedup**: Keep referenced tool calls when deduplicating websocket input IDs (`9c024540`).

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
