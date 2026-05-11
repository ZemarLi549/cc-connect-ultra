# 通用 Agent 安装与部署指南（cc-connect）

本文面向 `cc-connect` 项目，详细说明以下 4 类接入：

- Claude Code（原生 agent）
- Codex（原生 agent）
- OpenClaw（通过 ACP 通用适配）
- Hermes（企业控制平面，与 cc-connect 集成）

---

## 0. 通用前置条件

### 0.1 安装 cc-connect

你可以二选一：

```bash
# 方式 1：npm 安装
npm install -g cc-connect
```

```bash
# 方式 2：源码构建
git clone https://github.com/ZemarLi549/cc-connect-ultra.git
cd cc-connect-ultra
go build -o ./dist/cc-connect ./cmd/cc-connect
```

### 0.2 准备配置文件

```bash
mkdir -p ~/.cc-connect
cp config.example.toml ~/.cc-connect/config.toml
```

或使用当前目录配置：

```bash
cp config.example.toml ./config.toml
```

### 0.3 基础检查

```bash
cc-connect --version
```

---

## 1. Claude Code 安装、部署与配置

### 1.1 安装 Claude Code CLI

```bash
npm install -g @anthropic-ai/claude-code
claude --version
```

### 1.2 鉴权（任选其一）

- OAuth 登录（按 CLI 提示完成）。
- API Key 模式（在 `cc-connect` 的 provider 中配置 `api_key`）。

### 1.3 在 cc-connect 中配置 Claude Code 项目

将以下内容加入 `config.toml`（路径按你实际项目改）：

```toml
[[projects]]
name = "claude-demo"

[projects.agent]
type = "claudecode"

[projects.agent.options]
work_dir = "/path/to/your/repo"
mode = "default" # default | acceptEdits | plan | auto | bypassPermissions
# reasoning_effort = "high" # low | medium | high | max
# allowed_tools = ["Read", "Grep", "Glob", "Bash", "Edit", "Write"]
# disallowed_tools = ["WebSearch", "WebFetch"]

[[projects.platforms]]
type = "telegram"

[projects.platforms.options]
token = "your-telegram-bot-token"
```

如使用 API Provider，可加：

```toml
[[projects.agent.providers]]
name = "anthropic"
api_key = "${ANTHROPIC_API_KEY}"
```

### 1.4 启动与验证

```bash
cc-connect -config ~/.cc-connect/config.toml
```

在聊天平台发送：

- `/status`
- `/mode`
- `/new`

如果你是在 Claude Code 会话里启动 `cc-connect`，先执行：

```bash
unset CLAUDECODE && cc-connect
```

---

## 2. Codex 安装、部署与配置

### 2.1 安装 Codex CLI

```bash
npm install -g @openai/codex
codex --version
```

### 2.2 鉴权（任选其一）

- OAuth 模式：按 Codex CLI 登录流程完成（通常是 `codex login` 后按提示操作）。
- API Key 模式：在 provider 里填 `api_key`（OpenAI 或兼容网关）。

### 2.3 在 cc-connect 中配置 Codex 项目

```toml
[[projects]]
name = "codex-demo"

[projects.agent]
type = "codex"

[projects.agent.options]
work_dir = "/path/to/your/repo"
mode = "suggest" # suggest | auto-edit | full-auto | yolo
# model = "gpt-5.3-codex"
# reasoning_effort = "high" # minimal | low | medium | high | xhigh

[[projects.agent.providers]]
name = "openai"
api_key = "${OPENAI_API_KEY}"
# base_url = "https://api.openai.com/v1"
# model = "gpt-5.3-codex"

[[projects.platforms]]
type = "telegram"

[projects.platforms.options]
token = "your-telegram-bot-token"
```

### 2.4 启动与验证

```bash
cc-connect -config ~/.cc-connect/config.toml
```

聊天验证：

- `/status`
- `/reasoning`
- `/model`
- `/provider list`

---

## 3. OpenClaw 安装、部署与配置（ACP 方式）

`cc-connect` 中 OpenClaw 不是 `type="openclaw"`，而是通过通用 ACP 适配：

- `type = "acp"`
- `command = "openclaw"`
- `args = ["acp"]`

### 3.1 安装 OpenClaw CLI 与 Gateway

根据官方文档执行（仓库示例引用）：

- https://docs.openclaw.ai/cli/acp

完成后建议检查：

```bash
openclaw --version
```

### 3.2 准备 Gateway 连接信息

常见两种方式：

- 本地默认网关：`openclaw acp`
- 远端网关：`openclaw acp --url ... --token-file ...`

建议使用 `--token-file`，避免把 token 明文放在命令行参数中。

### 3.3 在 cc-connect 中配置 OpenClaw（ACP）

```toml
[[projects]]
name = "openclaw-acp"

[projects.agent]
type = "acp"

[projects.agent.options]
work_dir = "/path/to/your/repo"
command = "openclaw"
args = ["acp"]
display_name = "OpenClaw ACP"

[[projects.platforms]]
type = "telegram"

[projects.platforms.options]
token = "your-telegram-bot-token"
```

远端网关示例：

```toml
[projects.agent.options]
work_dir = "/path/to/your/repo"
command = "openclaw"
args = ["acp", "--url", "wss://gateway-host:18789", "--token-file", "/path/to/gateway.token"]
display_name = "OpenClaw ACP"
```

可选：绑定既有会话

```toml
[projects.agent.options]
command = "openclaw"
args = ["acp", "--session", "agent:main:main"]
```

### 3.4 启动与验证

```bash
cc-connect -config ~/.cc-connect/config.toml
```

聊天验证：

- `/status`
- `/new`
- `/list`

---

## 4. Hermes 部署与配置（控制平面集成）

说明：当前仓库没有 `type="hermes"` agent。  
Hermes 在本架构里是“企业控制平面”，与 `cc-connect` 运行时协同。

参考文档：

- `docs/hermes-cc-connect-topology.md`
- `docs/enterprise-platform-design.md`
- `docs/management-api.md`

### 4.1 推荐部署模型

- Hermes：负责租户/用户/空间/策略/Provider 编排。
- cc-connect：负责实际运行 agent 会话与消息路由。
- 每个空间（space）映射独立 `work_dir` 和会话边界，避免上下文串线。

### 4.2 打开 cc-connect 管理接口（供 Hermes 调用）

在 `config.toml` 中启用 `management`（可选同时启用 `webhook`）：

```toml
[management]
enabled = true
port = 9820
token = "your-mgmt-secret"
cors_origins = ["http://localhost:3000"]

[webhook]
enabled = true
port = 9111
token = "your-webhook-secret"
path = "/hook"
```

### 4.3 Hermes 对接 cc-connect 的最小流程

1. Hermes 维护 tenant/user/space/provider 元数据。
2. Hermes 通过 Management API 创建/更新项目配置（project -> space）。
3. Hermes 将执行请求转发给 cc-connect（可走管理接口或 webhook 触发）。
4. cc-connect 执行 agent，会话与 usage 回传 Hermes 聚合。

### 4.4 多空间配置建议

- 一个 space 对应一个 `[[projects]]`（或企业 API 抽象后的等价映射）。
- 每个 space 独立 `work_dir`。
- 不要让所有用户共享一个 agent 会话和同一个工作目录。

### 4.5 联调验证

先检查管理接口可达：

```bash
curl -H "Authorization: Bearer your-mgmt-secret" http://127.0.0.1:9820/api/v1/health
```

再让 Hermes 发起一次请求，确认：

- 目标 space 路由正确
- agent 正常回复
- usage 被正确记录

---

## 5. 常见问题排查（四类通用）

### 5.1 CLI 找不到

- 检查 `PATH` 是否包含 npm global bin。
- 使用绝对路径填 `command`（尤其 systemd/launchd 下）。

### 5.2 能启动但不回消息

- 先看 `cc-connect` 日志，把 `[log].level` 改成 `debug`。
- 确认平台 token/app_secret 正确。
- 确认项目 `work_dir` 存在且有读写权限。

### 5.3 OpenClaw ACP 连接失败

- 先独立执行 `openclaw acp` 验证 CLI 与网关。
- 远端模式优先 `--token-file`，确认文件权限和路径正确。

### 5.4 Hermes 模式下串会话

- 检查是否错误地让多个用户共用同一 `work_dir`/session_key。
- 按 space 做目录与会话隔离。
