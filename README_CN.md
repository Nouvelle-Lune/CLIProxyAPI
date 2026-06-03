# CLIProxyAPI 个人分支

[English](README.md) | 中文 | [日本語](README_JA.md)

本仓库是 [`router-for-me/CLIProxyAPI`](https://github.com/router-for-me/CLIProxyAPI) 的个人分支。用于在上游 CLIProxyAPI 基础上保持可复现的公开构建。

本分支仅供个人使用。按原样提供，维护者不对任何使用、部署、账户、计费、数据、合规或服务可用性后果承担任何保证、支持承诺或责任。

## 为什么存在这个分支

这个分支的主要目的是让 Claude Code 和其他 CLI AI 客户端在通过第三方 Anthropic 兼容、OpenAI 兼容、Gemini 兼容或混合提供商后端路由时，行为更加可预测。

实际上，这些客户端通常依赖微妙的协议行为：对话中的 system 消息、工具历史、thinking/reasoning 块、响应签名、模型后缀、重试行为和请求日志。代理在翻译或规范化这些字段时的微小差异可能会破坏提供商兼容性、使前缀缓存假设失效，或使调试变得困难。

本分支尽可能保持上游架构，但携带了对我的部署有用且更容易从源代码审计的实用变更。

## 与上游的结构差异

分支 README 与上游 README 是完全不同的文档。以下是各部分的对比：

| 部分 | 上游 | 分支 |
|------|------|------|
| **标题** | `CLI Proxy API` + 多语言链接 | `CLIProxyAPI 个人分支` |
| **赞助商** | 4 个赞助商的详细广告表格（PackyCode、AICodeMirror、BmoPlus、VisionCoder、APIKEY.FUN） | 无 |
| **功能特性** | 完整的功能列表（16 项） | 无 |
| **新手入门** | 指向 `help.router-for.me` 文档站 | 无 |
| **管理 API** | 指向外部文档 | 无 |
| **使用量统计** | CPA Usage Keeper、CPA-Manager-Plus 介绍 | 无 |
| **Amp CLI 支持** | 完整的 Amp 集成指南 | 无 |
| **SDK 文档** | 4 个 SDK 文档链接 | 无 |
| **贡献** | 标准贡献指南 | 无 |
| **谁与我们在一起** | 18 个基于 CLIProxyAPI 的社区项目 | 无 |
| **更多选择** | 4 个衍生项目 | 无 |
| **许可证** | MIT License | 无 |
| **为什么存在这个分支** | 无 | 分支的目的和动机 |
| **有什么不同** | 无 | 详细的差异说明（分支特有 + 上游合并变更） |
| **从源代码构建** | 无 | 完整的构建步骤 |
| **本地文件和密钥** | 无 | 安全提醒 |
| **上游** | 无 | 指向上游仓库 |

简而言之：分支 README 是一个精简的个人分支说明。去掉了上游所有的赞助商广告、功能介绍、生态项目列表和文档链接，替换为分支的目的、差异说明和构建指南。两者几乎没有重叠内容。

## 分支特有变更

本分支的近期变更重点：

- 在请求启用 Anthropic 的 `mid-conversation-system-*` beta 时，保留 Claude 对话中的 `role: "system"` 消息。
- 在请求转发到上游之前，去重重复的 Claude Code 任务提醒 system 消息。
- 当上游请求支持对话中 system 消息时，避免将 Claude Code 运行时 system 提醒不必要地移入顶层 Anthropic `system` 字段。
- 跨提供商桥接时保留内容转发。
- 改善跨提供商切换时 thinking 块、reasoning 历史、工具历史、响应模型和规范化消息的行为。
- 添加或携带关于响应 WebSocket 输入项去重、`count_tokens` auth 状态处理、模型后缀去除和提供商签名验证的兼容性修复。
- 保留请求日志和本地调试支持，用于验证代理发送的上游负载。
- 添加小型检查点辅助脚本用于本地开发快照。

## 携带的上游变更

截至最新的上游合并（`7bb7116e`），本分支还携带以下上游变更：

- **Claude 翻译协议修复**：OpenAI 工具消息合并以符合角色交替规则，通过 gjson/sjson 保留 JSON 键顺序以保持 KV 缓存稳定性，以及 Codex `web_search_call` 流式协议对齐（`364aa229`）。
- **运行时 auth 移除**：`Manager.Remove` 用于删除运行时 auth 和取消调度关联任务（`55440f0a`）。
- **Cloudflare 挑战重试**：对 403 Cloudflare 挑战使用渐进式限流冷却，而非硬性凭证挂起（`45f58d4f`、`77061aad`）。
- **Codex 修复**：避免重放孤立工具调用（`17af0891`）、处理非空 reasoning 和 content 项（`0e3c809c`）、缓存 reasoning 重放项（`603a08fc`）。
- **Gemini system role 规范化**：为 Gemini 提供商规范化消息级 system role（`68282c4a`）。
- **uTLS HTTP 客户端增强**：为受保护主机支持上下文感知 RoundTripper 以绕过 Cloudflare（`35ab084f`）。
- **Home auth 刷新修复**：解析 Home 刷新 auth 信封以正确使用刷新的 access token（`c9dc6bd6`）。
- **WebSocket 输入 ID 去重**：在去重 WebSocket 输入 ID 时保留引用的工具调用（`9c024540`）。

这不是上游 CLIProxyAPI 的通用替代品。这是一个带有针对性行为变更的个人工作分支。

## 从源代码构建

要求：

- Go 1.26 或更高版本
- Git

构建服务器二进制文件：

```bash
git clone https://github.com/Nouvelle-Lune/CLIProxyAPI.git
cd CLIProxyAPI
go mod download
go build -o cli-proxy-api ./cmd/server
```

创建本地配置并运行服务器：

```bash
cp config.example.yaml config.yaml
./cli-proxy-api --config config.yaml
```

开发时也可以不创建二进制文件直接运行：

```bash
go run ./cmd/server --config config.yaml
```

## 本地文件和密钥

不要提交本地运行时数据或密钥。本分支有意忽略本地部署文件，例如：

- `config.yaml`
- `cliproxyapi.conf`
- `.env`
- `auths/`
- `logs/`
- 本地构建的二进制文件
- 本地重建/重启脚本
- 临时工作文件

在发布或推送任何更改之前，请检查仓库中是否包含 API 密钥、OAuth 令牌、个人配置文件、请求日志和本地路径。

## 上游

原始项目：

```text
https://github.com/router-for-me/CLIProxyAPI
```

本分支应被视为个人下游变体，而非上游项目的官方文档。
