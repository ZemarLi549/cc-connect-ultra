# 技术架构设计文档 — AI 智能工作平台

> **版本**: v1.0
> **日期**: 2026-05-09
> **状态**: 待评审

---

## 1. 系统总体架构

### 1.1 架构分层

```
┌─────────────────────────────────────────────────────────────┐
│                       客户端层 (Client)                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ Web App  │  │ 飞书 Bot │  │ 钉钉 Bot │  │Slack Bot │   │
│  │ React    │  │          │  │          │  │          │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
├─────────────────────────────────────────────────────────────┤
│                       接入层 (Gateway)                       │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Nginx / Traefik                                      │  │
│  │  TLS 终止 · 速率限制 · 请求路由 · 静态资源             │  │
│  └──────────────────────────────────────────────────────┘  │
├───────────────────────────────────────────��─────────────────┤
│                    应用服务层 (Application)                   │
│                                                              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ Auth Service│  │ API Service│  │ Agent      │           │
│  │ 认证授权    │  │ 业务 API   │  │ Service    │           │
│  │            │  │            │  │ AI 编排    │           │
│  └────────────┘  └────────────┘  └────────────┘           │
│                                                              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐           │
│  │ Scheduler  │  │ Plugin     │  │ File       │           │
│  │ Service    │  │ Manager    │  │ Service    │           │
│  │ 定时调度    │  │ 插件生命周期│  │ 文件管理   │           │
│  └────────────┘  └────────────┘  └────────────┘           │
├─────────────────────────────────────────────────────────────┤
│                    AI 引擎层 (AI Engine)                      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Agent Runtime                                       │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐          │  │
│  │  │ Prompt   │  │ Skill    │  │ MCP      │          │  │
│  │  │ Manager  │  │ Engine   │  │ Client   │          │  │
│  │  └──────────┘  └──────────┘  └──────────┘          │  │
│  └──────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                    MCP 工具层 (MCP Tools)                     │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │ Task MCP │ │ Object   │ │ Schedule │ │ File MCP │     │
│  │ Server   │ │ MCP      │ │ MCP      │ │ Server   │     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘     │
├─────────────────────────────────────────────────────────────┤
│                    执行沙箱层 (Sandbox)                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │
│  │Session 1 │ │Session 2 │ │Session 3 │ │Session N │     │
│  │Container │ │Container │ │Container │ │Container │     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘     │
│  Docker / containerd · seccomp · cgroups · networkns       │
├─────────────────────────────────────────────────────────────┤
│                    数据层 (Data)                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │ PostgreSQL   │  │ Redis        │  │ Object      │    │
│  │ 主数据库     │  │ 缓存/会话    │  │ Storage     │    │
│  │ JSONB        │  │              │  │ (S3/MinIO)  │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                    基础设施层 (Infrastructure)                │
│  Docker · Kubernetes · Prometheus · Grafana · ELK           │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 技术选型

| 层级 | 技术 | 版本 | 选型理由 |
|---|---|---|---|
| **后端语言** | Go | 1.22+ | 高性能、并发友好、部署简单（单二进制） |
| **AI 引擎** | Python | 3.11+ | Claude SDK 生态、AI/ML 库丰富 |
| **前端框架** | React | 18+ | 组件化、生态丰富、团队熟悉 |
| **构建工具** | Vite | 5+ | 快速 HMR、优秀构建性能 |
| **UI 框架** | Ant Design | 5+ | 企业级组件丰富、定制灵活 |
| **数据库** | PostgreSQL | 16+ | JSONB 支持、性能优秀、开源 |
| **缓存** | Redis | 7+ | 会话缓存、提示词缓存、分布式锁 |
| **对象存储** | MinIO / S3 | - | 文件存储、可替换 |
| **消息队列** | NATS | 2+ | 轻量高性能、适合事件驱动 |
| **容器** | Docker | 24+ | 沙箱隔离 |
| **编排** | Kubernetes | 1.28+ | 容器编排、自动扩缩 |
| **LLM SDK** | Claude Agent SDK | latest | 官方 Agent 框架 |
| **API 框架 (Go)** | Gin | 1.9+ | 高性能 HTTP 框架 |
| **ORM (Go)** | GORM | 2.0+ | Go 最流行的 ORM |
| **AI 框架** | FastAPI | 0.100+ | Python 异步 API 框架 |
| **WebSocket** | Socket.IO | 4+ | 实时双向通信 |
| **监控** | Prometheus + Grafana | - | 指标采集和可视化 |
| **日志** | ELK Stack | - | 日志收集和分析 |
| **CI/CD** | GitHub Actions | - | 自动化构建部署 |

---

## 2. 后端服务详细设计

### 2.1 Go 后端服务 (elevo-server)

#### 2.1.1 项目结构

```
elevo-server/
├── cmd/
│   └── server/
│       └── main.go                 # 入口
├── internal/
│   ├── config/
│   │   └── config.go               # 配置管理
│   ├── middleware/
│   │   ├── auth.go                 # JWT 认证中间件
│   │   ├── cors.go                 # CORS
│   │   ├── rate_limit.go           # 速率限制
│   │   ├── tenant.go               # 多租户识别
│   │   ├── audit.go                # 审计日志
│   │   └── recovery.go             # 异常恢复
│   ├── handler/
│   │   ├── auth.go                 # 认证接口
│   │   ├── workspace.go            # 工作空间接口
│   │   ├── task.go                 # 任务接口
│   │   ├── business_object.go      # 业务对象接口
│   │   ├── scheduled_job.go        # 定时任务接口
│   │   ├── file.go                 # 文件接口
│   │   ├── member.go               # 成员接口
│   │   └── chat.go                 # 聊天接口
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── workspace_service.go
│   │   ├── task_service.go
│   │   ├── object_service.go
│   │   ├── scheduler_service.go
│   │   ├── file_service.go
│   │   └── member_service.go
│   ├── repository/
│   │   ├── task_repo.go
│   │   ├── object_repo.go
│   │   ├── workspace_repo.go
│   │   └── user_repo.go
│   ├── model/
│   │   ├── task.go
│   │   ├── business_object.go
│   │   ├── scheduled_job.go
│   │   ├── workspace.go
│   │   ├── user.go
│   │   └── session.go
│   ├── mcp/
│   │   ├── client.go               # MCP Client 实现
│   │   ├── server.go               # MCP Server 注册
│   │   └── tools/
│   │       ├── task_tools.go       # 任务相关 MCP 工具
│   │       ├── object_tools.go     # 业务对象 MCP 工具
│   │       ├── file_tools.go       # 文件 MCP 工具
│   │       └── schedule_tools.go   # 定时任务 MCP 工具
│   └── sandbox/
│       ├── manager.go              # 沙箱生命周期管理
│       ├── container.go            # Docker 容器操作
│       └── policy.go               # 安全策略 (seccomp, cgroups)
├── pkg/
│   ├── errors/                     # 统一错误码
│   ├── response/                   # 统一响应格式
│   ├── pagination/                 # 分页工具
│   └── validator/                  # 参数校验
├── migrations/
│   ├── 001_create_tenants.sql
│   ├── 002_create_users.sql
│   ├── 003_create_workspaces.sql
│   ├── 004_create_tasks.sql
│   ├── 005_create_business_objects.sql
│   ├── 006_create_scheduled_jobs.sql
│   └── 007_create_sessions.sql
├── api/
│   └── openapi.yaml                # OpenAPI 3.0 规范
├── deployments/
│   ├── docker/
│   │   └── Dockerfile
│   └── k8s/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── configmap.yaml
├── go.mod
├── go.sum
└── Makefile
```

#### 2.1.2 核心配置

```yaml
# config.yaml
server:
  host: 0.0.0.0
  port: 8080
  mode: release  # debug / release

database:
  host: postgres
  port: 5432
  name: elevo
  user: elevo
  password: ${DB_PASSWORD}
  max_open_conns: 50
  max_idle_conns: 10
  ssl_mode: require

redis:
  host: redis
  port: 6379
  password: ${REDIS_PASSWORD}
  db: 0

auth:
  jwt_secret: ${JWT_SECRET}
  jwt_expiry: 24h
  refresh_expiry: 168h  # 7 days

sandbox:
  docker_socket: /var/run/docker.sock
  image: elevo/sandbox:latest
  cpu_limit: 2
  memory_limit: 4g
  disk_limit: 10g
  network_disabled: true
  session_timeout: 30m
  max_containers: 100

ai:
  anthropic_api_key: ${ANTHROPIC_API_KEY}
  model: claude-sonnet-4-20250514
  max_tokens: 8192
  prompt_cache_ttl: 300s
  temperature: 0.7

storage:
  provider: minio  # minio / s3
  endpoint: http://minio:9000
  bucket: elevo-files
  access_key: ${S3_ACCESS_KEY}
  secret_key: ${S3_SECRET_KEY}

nats:
  url: nats://nats:4222

logging:
  level: info
  format: json
```

#### 2.1.3 统一响应格式

```go
// pkg/response/response.go

type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
    Page       int   `json:"page,omitempty"`
    PageSize   int   `json:"page_size,omitempty"`
    Total      int64 `json:"total,omitempty"`
    TotalPages int   `json:"total_pages,omitempty"`
}

// 成功响应
func Success(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, Response{
        Code:    0,
        Message: "success",
        Data:    data,
    })
}

// 分页响应
func SuccessWithPagination(c *gin.Context, data interface{}, meta Meta) {
    c.JSON(http.StatusOK, Response{
        Code:    0,
        Message: "success",
        Data:    data,
        Meta:    &meta,
    })
}

// 错误响应
func Error(c *gin.Context, code int, message string) {
    c.JSON(code, Response{
        Code:    code,
        Message: message,
    })
}

// 分页工具
type Pagination struct {
    Page     int `form:"page" binding:"omitempty,min=1"`
    PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

func (p *Pagination) Normalize() {
    if p.Page == 0 {
        p.Page = 1
    }
    if p.PageSize == 0 {
        p.PageSize = 20
    }
}

func (p *Pagination) Offset() int {
    return (p.Page - 1) * p.PageSize
}
```

#### 2.1.4 多租户中间件

```go
// internal/middleware/tenant.go

func TenantMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        workspaceID := c.GetHeader("X-Workspace-Id")
        if workspaceID == "" {
            workspaceID = c.Param("workspace_id")
        }
        if workspaceID == "" {
            Error(c, http.StatusBadRequest, "workspace_id is required")
            c.Abort()
            return
        }

        // 验证 workspace 存在且用户有权限
        userID := c.GetString("user_id")
        if !checkWorkspaceAccess(userID, workspaceID) {
            Error(c, http.StatusForbidden, "access denied to workspace")
            c.Abort()
            return
        }

        c.Set("workspace_id", workspaceID)
        c.Next()
    }
}
```

#### 2.1.5 任务服务核心逻辑

```go
// internal/service/task_service.go

type TaskService struct {
    repo   repository.TaskRepository
    userRepo repository.UserRepository
}

func (s *TaskService) CreateTask(ctx context.Context, wsID string, req *CreateTaskRequest) (*Task, error) {
    task := &model.Task{
        WorkspaceID: wsID,
        Title:       req.Title,
        Description: req.Description,
        Type:        req.Type,
        Priority:    req.Priority,
        DueDate:     req.DueDate,
        StartDate:   req.StartDate,
        ParentID:    req.ParentID,
    }

    // 解析 assignee_id
    switch req.AssigneeID {
    case "me":
        task.AssigneeID = ctx.Value("user_id").(string)
    case "none", "":
        task.AssigneeID = nil
    default:
        // 验证用户存在且为工作空间成员
        if !s.userRepo.IsMember(ctx, wsID, req.AssigneeID) {
            return nil, ErrUserNotMember
        }
        task.AssigneeID = &req.AssigneeID
    }

    // 验证父任务存在
    if req.ParentID != nil {
        parent, err := s.repo.GetByID(ctx, wsID, *req.ParentID)
        if err != nil {
            return nil, ErrParentNotFound
        }
        if parent.Type != "goal" {
            return nil, ErrInvalidParentType
        }
    }

    if err := s.repo.Create(ctx, task); err != nil {
        return nil, err
    }

    // 创建标签
    for _, tag := range req.Tags {
        s.repo.AddTag(ctx, task.ID, tag)
    }

    return task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, wsID string, filter *TaskFilter) ([]*Task, int64, error) {
    // 构建查询条件
    query := s.repo.Query(ctx, wsID)

    if filter.Status != nil {
        query = query.WithStatus(filter.Status...)
    }
    if filter.Type != nil {
        query = query.WithType(filter.Type...)
    }
    if filter.Priority != nil {
        query = query.WithPriority(filter.Priority...)
    }
    if filter.AssigneeID != "" {
        query = query.WithAssignee(filter.AssigneeID)
    }
    if filter.ParentID == "null" {
        query = query.TopLevelOnly()
    }
    if filter.Keyword != "" {
        query = query.WithKeyword(filter.Keyword)
    }
    if filter.DueDate != nil {
        if filter.DueDate.Overdue {
            query = query.Overdue()
        }
        if filter.DueDate.Today {
            query = query.DueToday()
        }
        if filter.DueDate.ThisWeek {
            query = query.DueThisWeek()
        }
    }
    if filter.Tags != nil {
        query = query.WithTags(filter.Tags.Include, filter.Tags.Match)
    }

    return query.
        OrderBy(filter.SortBy, filter.SortOrder).
        Paginate(filter.Page, filter.PageSize).
        Execute()
}
```

### 2.2 AI 引擎服务 (elevo-ai)

#### 2.2.1 项目结构

```
elevo-ai/
├── app/
│   ├── __init__.py
│   ├── main.py                    # FastAPI 入口
│   ├── config.py                  # 配置
│   └── dependencies.py            # 依赖注入
├── agent/
│   ├── __init__.py
│   ├── runtime.py                 # Agent 运行时
│   ├── prompt_builder.py          # 系统提示词构建器
│   ├── skill_engine.py            # 技能引擎
│   └── context_manager.py         # 上下文管理
├── mcp/
│   ├── __init__.py
│   ├── client.py                  # MCP 客户端
│   ├── protocol.py                # MCP 协议定义
│   └── registry.py                # 工具注册表
├── skills/
│   ├── __init__.py
│   ├── base.py                    # 技能基类
│   ├── task_skills.py             # 任务管理技能
│   ├── scheduler_skills.py        # 定时任务技能
│   └── dashboard_skills.py        # 仪表板技能
├── integrations/
│   ├── __init__.py
│   ├── anthropic.py               # Claude API 集成
│   ├── feishu.py                  # 飞书集成
│   ├── dingtalk.py                # 钉钉集成
│   └── slack.py                   # Slack 集成
├── prompts/
│   ├── system_prompt.txt          # 系统提示词模板
│   ├── role_definition.txt        # 角色定义
│   └── security_rules.txt         # 安全规则
├── schemas/
│   ├── ask_human.json             # ask_human 参数 Schema
│   └── present_result.json        # present_result 参数 Schema
├── tests/
│   ├── test_agent.py
│   ├── test_skill_engine.py
│   └── test_mcp_client.py
├── requirements.txt
├── Dockerfile
└── pyproject.toml
```

#### 2.2.2 Agent 运行时

```python
# agent/runtime.py

import anthropic
from anthropic import Anthropic
from agent.prompt_builder import PromptBuilder
from agent.skill_engine import SkillEngine
from agent.context_manager import ContextManager
from mcp.client import MCPClient

class AgentRuntime:
    def __init__(self, config):
        self.client = Anthropic(api_key=config.anthropic_api_key)
        self.model = config.model
        self.prompt_builder = PromptBuilder()
        self.skill_engine = SkillEngine(config.skills_dir)
        self.context_manager = ContextManager(config.cache_ttl)
        self.mcp_client = MCPClient(config.mcp_servers)

    async def process_message(
        self,
        session_id: str,
        workspace_id: str,
        user_id: str,
        message: str,
        conversation_history: list[dict],
    ) -> str:
        """处理用户消息，返回 AI 回复"""

        # 1. 构建系统提示词
        system_prompt = self.prompt_builder.build(
            workspace_id=workspace_id,
            user_id=user_id,
            session_id=session_id,
        )

        # 2. 检查技能匹配
        matched_skill = self.skill_engine.match(message)
        if matched_skill:
            # 加载技能详细说明
            skill_instructions = self.skill_engine.load(matched_skill)
            system_prompt += f"\n\n## Active Skill: {matched_skill}\n{skill_instructions}"

        # 3. 获取 MCP 工具列表
        tools = self.mcp_client.get_tools()

        # 4. 调用 Claude API
        response = self.client.messages.create(
            model=self.model,
            max_tokens=8192,
            system=system_prompt,
            messages=conversation_history + [{"role": "user", "content": message}],
            tools=tools,
        )

        # 5. 处理工具调用
        while response.stop_reason == "tool_use":
            tool_results = []
            for block in response.content:
                if block.type == "tool_use":
                    result = await self.mcp_client.execute_tool(
                        tool_name=block.name,
                        tool_input=block.input,
                        workspace_id=workspace_id,
                    )
                    tool_results.append({
                        "type": "tool_result",
                        "tool_use_id": block.id,
                        "content": result,
                    })

            # 将工具结果送回模型继续
            response = self.client.messages.create(
                model=self.model,
                max_tokens=8192,
                system=system_prompt,
                messages=conversation_history + [
                    {"role": "user", "content": message},
                    {"role": "assistant", "content": response.content},
                    {"role": "user", "content": tool_results},
                ],
                tools=tools,
            )

        # 6. 提取文本回复
        return self._extract_text(response)

    def _extract_text(self, response) -> str:
        """从响应中提取纯文本"""
        text_parts = []
        for block in response.content:
            if block.type == "text":
                text_parts.append(block.text)
        return "\n".join(text_parts)
```

#### 2.2.3 提示词构建器

```python
# agent/prompt_builder.py

class PromptBuilder:
    def build(self, workspace_id: str, user_id: str, session_id: str) -> str:
        parts = []

        # 1. 角色定义
        parts.append(self._load_template("role_definition.txt"))

        # 2. 行为规则
        parts.append(self._load_template("security_rules.txt"))

        # 3. 技能列表
        parts.append(self._build_skill_list())

        # 4. 工作空间配置
        parts.append(f"""
## Workspace Configuration

### Session Context
- Your Session ID: `{session_id}`
- Current working directory: workspace root
- All relative paths are resolved from workspace root

### Directory Structure
```
./
├── .elevo/              # System metadata (read-only)
├── CLAUDE.md            # AI agent guidance (read-only)
├── AGENTS.md            # Skills/agents definition (read-only)
├── knowledge/           # Workspace knowledge base
├── .session/{session_id}/  # Your session directory (writable)
│   ├── context.json     # Session context
│   ├── context/         # @mention context files
│   └── documents/       # Uploaded files
└── *.md, *.json, ...    # User persistent files
```

### First Action
Before calling ANY MCP tools, read `.session/{session_id}/context.json` to get `workspace_id`.

### Access Rules
- Writable: `.session/{session_id}/` and subdirectories
- Read-only: `.elevo/`, `CLAUDE.md`, `AGENTS.md`
- Forbidden: Other session directories, absolute paths
""")

        # 5. 语言规则
        parts.append("""
## Language Adaptation
All visible content MUST use the same language as the user's current input.
- Chinese input → All output in Chinese
- English input → All output in English
""")

        return "\n\n".join(parts)

    def _load_template(self, filename: str) -> str:
        with open(f"prompts/{filename}", "r") as f:
            return f.read().strip()

    def _build_skill_list(self) -> str:
        # 动态扫描 skills/ 目录生成技能列表
        ...
```

#### 2.2.4 技能引擎

```python
# agent/skill_engine.py

import re
import os
import yaml

class SkillEngine:
    def __init__(self, skills_dir: str):
        self.skills_dir = skills_dir
        self.skills = self._load_skills()

    def _load_skills(self) -> dict:
        """加载所有技能定义"""
        skills = {}
        for filename in os.listdir(self.skills_dir):
            if filename.endswith(".yaml"):
                with open(os.path.join(self.skills_dir, filename)) as f:
                    skill = yaml.safe_load(f)
                    skills[skill["name"]] = skill
        return skills

    def match(self, message: str) -> str | None:
        """根据用户消息匹配技能"""
        msg_lower = message.lower().strip()

        for skill_name, skill in self.skills.items():
            for trigger in skill.get("triggers", []):
                # 支持正则匹配
                if trigger.startswith("^") or trigger.endswith("$"):
                    if re.search(trigger, msg_lower, re.IGNORECASE):
                        return skill_name
                else:
                    if trigger.lower() in msg_lower:
                        return skill_name

        return None

    def load(self, skill_name: str) -> str:
        """加载技能详细说明"""
        skill = self.skills.get(skill_name)
        if not skill:
            return ""

        instruction_file = os.path.join(
            self.skills_dir,
            skill_name,
            "instructions.md"
        )
        if os.path.exists(instruction_file):
            with open(instruction_file) as f:
                return f.read()

        return skill.get("description", "")
```

#### 2.2.5 MCP 客户端

```python
# mcp/client.py

import httpx
from mcp.protocol import MCPRequest, MCPResponse

class MCPClient:
    def __init__(self, servers: dict):
        """
        servers: {
            "task-mcp": {"url": "http://localhost:8001", "tools": [...]},
            "file-mcp": {"url": "http://localhost:8002", "tools": [...]},
        }
        """
        self.servers = servers
        self.http_client = httpx.AsyncClient(timeout=30.0)

    def get_tools(self) -> list[dict]:
        """聚合所有 MCP Server 的工具定义"""
        all_tools = []
        for server_name, server_config in self.servers.items():
            for tool in server_config.get("tools", []):
                tool["_server"] = server_name
                all_tools.append(tool)
        return all_tools

    async def execute_tool(self, tool_name: str, tool_input: dict, workspace_id: str) -> str:
        """执行工具调用"""
        # 找到工具所属的 server
        server_name = self._find_server(tool_name)
        if not server_name:
            return f"Error: Tool '{tool_name}' not found"

        server_url = self.servers[server_name]["url"]

        # 确保 workspace_id 被注入
        tool_input["workspace_id"] = workspace_id

        # 调用 MCP Server
        try:
            response = await self.http_client.post(
                f"{server_url}/mcp/tools/{tool_name}/execute",
                json=MCPRequest(
                    tool_name=tool_name,
                    input=tool_input,
                ).model_dump(),
            )
            result = MCPResponse(**response.json())
            return result.content
        except Exception as e:
            return f"Error executing tool '{tool_name}': {str(e)}"

    def _find_server(self, tool_name: str) -> str | None:
        for server_name, server_config in self.servers.items():
            for tool in server_config.get("tools", []):
                if tool["name"] == tool_name:
                    return server_name
        return None
```

---

## 3. 数据库设计

### 3.1 ER 关系图

```
┌──────────────┐       ┌──────────────┐       ┌──────────────────┐
│   tenants    │       │    users     │       │ workspace_members │
├──────────────┤       ├──────────────┤       ├──────────────────┤
│ id (PK)      │──┐    │ id (PK)      │──┐    │ id (PK)          │
│ name         │  │    │ tenant_id(FK)│──┘    │ workspace_id(FK) │──┐
│ config       │  └───►│ email        │       │ user_id (FK)     │──┤
│ created_at   │       │ name         │       │ role             │  │
└──────────────┘       │ avatar       │       └──────────────────┘  │
                       └──────────────┘                             │
                              │                                     │
┌──────────────┐              │              ┌──────────────┐      │
│  workspaces  │◄─────────────┘              │    tasks     │      │
├──────────────┤                             ├──────────────┤      │
│ id (PK)      │─────────────────────────────►│ workspace_id │◄─────┘
│ tenant_id(FK)│                             │ id (PK)      │
│ name         │                             │ title        │
│ manifest     │                             │ type         │
│ created_at   │                             │ status       │
└──────────────┘                             │ priority     │
       │                                     │ assignee_id  │──┐
       │              ┌──────────────┐       │ parent_id(FK)│  │
       │              │ task_tags    │       │ due_date     │  │
       │              ├──────────────┤       └──────┬───────┘  │
       │              │ task_id(FK)  │◄──────────────┘          │
       │              │ tag_name     │       │ self-referencing  │
       │              └──────────────┘       │ (parent_id)      │
       │                                     │                  │
       │              ┌──────────────┐       │                  │
       │              │task_comments │       │                  │
       │              ├──────────────┤       │                  │
       │              │ task_id(FK)  │◄──────┘                  │
       │              │ user_id (FK) │──────────────────────────┘
       │              │ content      │  (users.id)
       │              └──────────────┘
       │
       │              ┌──────────────────┐   ┌──────────────────┐
       │              │ business_objects  │   │ object_instances │
       │              ├──────────────────┤   ├──────────────────┤
       │              │ workspace_id (FK)│   │ workspace_id (FK)│
       │              │ name             │   │ object_type      │──► business_objects.name
       │              │ schema (JSONB)   │   │ data (JSONB)     │
       │              │ state_machine    │   │ current_state    │
       │              └──────────────────┘   └──────────────────┘
       │
       │              ┌──────────────────┐
       │              │ scheduled_jobs   │
       │              ├──────────────────┤
       │              │ workspace_id(FK) │
       │              │ name             │
       │              │ prompt           │
       │              │ schedule_type    │
       │              │ schedule_config  │
       │              │ enabled          │
       │              │ next_run_at      │
       └──────────────┘

       ┌──────────────┐
       │   sessions   │
       ├──────────────┤
       │ id (PK)      │
       │ workspace_id │
       │ user_id      │
       │ status       │
       │ container_id │
       │ created_at   │
       │ last_active  │
       └──────────────┘
```

### 3.2 完整 SQL Schema

```sql
-- migrations/001_create_tenants.sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(200) NOT NULL,
    config     JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tenants_name ON tenants(name);

-- migrations/002_create_users.sql
CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email      VARCHAR(255) UNIQUE,
    name       VARCHAR(200) NOT NULL,
    avatar     VARCHAR(500),
    role       VARCHAR(50) DEFAULT 'member',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);

-- migrations/003_create_workspaces.sql
CREATE TABLE workspaces (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(200) NOT NULL,
    manifest   JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_workspaces_tenant ON workspaces(tenant_id);

CREATE TABLE workspace_members (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         VARCHAR(50) DEFAULT 'member',
    joined_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (workspace_id, user_id)
);

CREATE INDEX idx_wm_workspace ON workspace_members(workspace_id);
CREATE INDEX idx_wm_user ON workspace_members(user_id);

-- migrations/004_create_tasks.sql
CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    task_number     VARCHAR(50),
    title           VARCHAR(500) NOT NULL,
    description     TEXT DEFAULT '',
    type            VARCHAR(20) NOT NULL DEFAULT 'task',
    status          VARCHAR(20) NOT NULL DEFAULT 'todo',
    priority        VARCHAR(10) NOT NULL DEFAULT 'medium',
    assignee_id     UUID REFERENCES users(id),
    created_by      UUID REFERENCES users(id),
    parent_id       UUID REFERENCES tasks(id) ON DELETE SET NULL,
    due_date        TIMESTAMPTZ,
    start_date      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    is_overdue      BOOLEAN GENERATED ALWAYS AS (
        due_date IS NOT NULL
        AND due_date < NOW()
        AND status NOT IN ('done', 'cancelled')
    ) STORED,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tasks_workspace ON tasks(workspace_id);
CREATE INDEX idx_tasks_status ON tasks(workspace_id, status);
CREATE INDEX idx_tasks_assignee ON tasks(workspace_id, assignee_id);
CREATE INDEX idx_tasks_type ON tasks(workspace_id, type);
CREATE INDEX idx_tasks_priority ON tasks(workspace_id, priority);
CREATE INDEX idx_tasks_due ON tasks(workspace_id, due_date);
CREATE INDEX idx_tasks_parent ON tasks(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_tasks_created_by ON tasks(workspace_id, created_by);
CREATE INDEX idx_tasks_search ON tasks USING GIN(to_tsvector('simple', title || ' ' || COALESCE(description, '')));

CREATE TABLE task_tags (
    task_id  UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tag_name VARCHAR(100) NOT NULL,
    PRIMARY KEY (task_id, tag_name)
);

CREATE INDEX idx_tags_name ON task_tags(tag_name);

CREATE TABLE task_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id),
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_comments_task ON task_comments(task_id);

-- 任务自动编号序列（每个工作空间独立）
CREATE SEQUENCE task_number_seq;

-- migrations/005_create_business_objects.sql
CREATE TABLE business_objects (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name          VARCHAR(200) NOT NULL,
    description   TEXT DEFAULT '',
    schema_def    JSONB NOT NULL DEFAULT '{}',
    state_machine JSONB DEFAULT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (workspace_id, name)
);

CREATE INDEX idx_bo_workspace ON business_objects(workspace_id);
CREATE INDEX idx_bo_search ON business_objects USING GIN(to_tsvector('simple', name || ' ' || COALESCE(description, '')));

CREATE TABLE object_instances (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    object_type   VARCHAR(200) NOT NULL,
    data          JSONB NOT NULL DEFAULT '{}',
    current_state VARCHAR(100),
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_oi_workspace ON object_instances(workspace_id);
CREATE INDEX idx_oi_type ON object_instances(workspace_id, object_type);
CREATE INDEX idx_oi_state ON object_instances(workspace_id, current_state);
CREATE INDEX idx_oi_data ON object_instances USING GIN(data jsonb_path_ops);

-- migrations/006_create_scheduled_jobs.sql
CREATE TABLE scheduled_jobs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id         UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name                 VARCHAR(100) NOT NULL,
    prompt               VARCHAR(5000) NOT NULL,
    schedule_type        VARCHAR(10) NOT NULL,
    schedule_config      JSONB NOT NULL,
    timeout_seconds      INTEGER NOT NULL DEFAULT 300 CHECK (timeout_seconds BETWEEN 60 AND 1800),
    enabled              BOOLEAN NOT NULL DEFAULT true,
    next_run_at          TIMESTAMPTZ,
    last_run_at          TIMESTAMPTZ,
    last_run_status      VARCHAR(20),
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    pause_reason         TEXT,
    created_at           TIMESTAMPTZ DEFAULT NOW(),
    updated_at           TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sj_workspace ON scheduled_jobs(workspace_id);
CREATE INDEX idx_sj_next_run ON scheduled_jobs(next_run_at) WHERE enabled = true;
CREATE INDEX idx_sj_enabled ON scheduled_jobs(workspace_id, enabled);

-- migrations/007_create_sessions.sql
CREATE TABLE sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(20) NOT NULL DEFAULT 'active',
    container_id VARCHAR(200),
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    last_active  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sessions_workspace ON sessions(workspace_id);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_status ON sessions(status);

-- 审计日志表
CREATE TABLE audit_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id   UUID REFERENCES sessions(id) ON DELETE SET NULL,
    action       VARCHAR(100) NOT NULL,
    resource     VARCHAR(100),
    resource_id  UUID,
    details      JSONB DEFAULT '{}',
    ip_address   INET,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_workspace ON audit_logs(workspace_id);
CREATE INDEX idx_audit_user ON audit_logs(user_id);
CREATE INDEX idx_audit_time ON audit_logs(created_at);
CREATE INDEX idx_audit_action ON audit_logs(action);
```

---

## 4. 沙箱安全设计

### 4.1 Docker 沙箱配置

```dockerfile
# sandbox/Dockerfile
FROM python:3.11-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 创建非 root 用户
RUN useradd -m -s /bin/bash sandbox
USER sandbox
WORKDIR /home/sandbox/project

# 默认命令保持运行
CMD ["sleep", "infinity"]
```

```python
# internal/sandbox/container.py

class SandboxManager:
    """管理会话沙箱容器的创建和销毁"""

    def __init__(self, config):
        self.client = docker.from_env()
        self.config = config

    def create_container(self, session_id: str, workspace_path: str) -> str:
        """为会话创建沙箱容器"""
        container = self.client.containers.run(
            image=self.config.sandbox_image,
            name=f"sandbox-{session_id[:12]}",
            detach=True,
            mem_limit=self.config.memory_limit,      # 4g
            cpu_quota=self.config.cpu_limit * 100000, # 200000 = 2 cores
            storage_opt={"size": self.config.disk_limit}, # 10g
            network_disabled=True,
            read_only=False,
            security_opt=[
                "no-new-privileges:true",
                f"seccomp={self._seccomp_profile_path()}",
            ],
            pids_limit=256,
            ulimits=[
                {"Name": "nofile", "Soft": 1024, "Hard": 1024},
                {"Name": "nproc", "Soft": 256, "Hard": 256},
            ],
            volumes={
                # 工作空间根目录 - 只读
                f"{workspace_path}": {
                    "bind": "/home/sandbox/project",
                    "mode": "ro",
                },
                # .elevo/ - 只读（已在上面包含）
                # 会话目录 - 读写
                f"{workspace_path}/.session/{session_id}": {
                    "bind": f"/home/sandbox/project/.session/{session_id}",
                    "mode": "rw",
                },
            },
        )
        return container.id

    def destroy_container(self, container_id: str):
        """销毁沙箱容器"""
        try:
            container = self.client.containers.get(container_id)
            container.remove(force=True)
        except docker.errors.NotFound:
            pass

    def _seccomp_profile_path(self) -> str:
        """返回 seccomp 安全配置文件路径"""
        return "/etc/elevo/seccomp-default.json
```

### 4.2 Seccomp 安全配置

```json
// seccomp-default.json
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "defaultErrnoRet": 1,
  "architectures": ["SCMP_ARCH_X86_64"],
  "syscalls": [
    {"names": ["read", "write", "close", "fstat", "mmap", "mprotect",
               "brk", "ioctl", "access", "pipe", "dup2", "getpid",
               "socket", "connect", "sendto", "recvfrom", "clone",
               "fork", "execve", "exit", "wait4", "openat", "fstatat",
               "newfstatat", "getdents64", "lseek", "clock_gettime",
               "futex", "set_tid_address", "rseq", "prlimit64",
               "pread64", "pwrite64", "getrandom", "sysinfo"],
     "action": "SCMP_ACT_ALLOW"},
    {"names": ["clone", "fork", "vfork"],
     "action": "SCMP_ACT_ALLOW",
     "args": [{"index": 0, "op": "SCMP_CMP_EQ", "val": 0}]},
    {"names": ["execve", "execveat"],
     "action": "SCMP_ACT_ERRNO",
     "errnoRet": 13}
  ]
}
```

### 4.3 危险命令拦截

```python
# internal/sandbox/policy.go

var ForbiddenPatterns = []struct {
    Pattern     string
    Description string
    Severity    string
}{
    {`rm\s+-rf\s+/`, "删除根目录", "critical"},
    {`>\s*/dev/sd`, "直接写块设备", "critical"},
    {`mkfs\.`, "格式化文件系统", "critical"},
    {`:(){ :\|:& };:`, "Fork 炸弹", "critical"},
    {`while\s+true.*do`, "无限循环", "high"},
    {`dd\s+if=`, "DD 攻击", "high"},
    {`nc\s+-[el]`, "网络工具", "high"},
    {`ping\s+-f`, "Flood Ping", "high"},
    {`chmod\s+777\s+/`, "权限提升", "high"},
    {`curl.*\|\s*bash`, "远程代码执行", "critical"},
    {`wget.*\|\s*sh`, "远程代码执行", "critical"},
}
```

---

## 5. 前端设计

### 5.1 React 项目结构

```
elevo-web/
├── src/
│   ├── app/
│   │   ├── layout.tsx               # 全局布局
│   │   ├── providers.tsx            # Context Providers
│   │   └── routes.tsx               # 路由配置
│   ├── pages/
│   │   ├── chat/
│   │   │   ├── ChatPage.tsx         # 聊天页面
│   │   │   ├── MessageBubble.tsx    # 消息气泡
│   │   │   ├── ChoiceCard.tsx       # 选择卡片
│   │   │   ├── FormCard.tsx         # 表单卡片
│   │   │   └── ResourceCard.tsx     # 资源卡片
│   │   ├── tasks/
│   │   │   ├── TaskListPage.tsx     # 任务列表
│   │   │   ├── TaskDetailPage.tsx   # 任务详情
│   │   │   ├── TaskFilters.tsx      # 任务筛选器
│   │   │   └── components/
│   │   │       ├── TaskCard.tsx
│   │   │       ├── StatusBadge.tsx
│   │   │       ├── PriorityTag.tsx
│   │   │       └── TaskForm.tsx
│   │   ├── dashboard/
│   │   │   └── DashboardPage.tsx    # 工作台
│   │   ├── objects/
│   │   │   ├── ObjectListPage.tsx   # 业务对象列表
│   │   │   └── ObjectDetailPage.tsx
│   │   ├── schedules/
│   │   │   └── ScheduleListPage.tsx
│   │   ├── members/
│   │   │   └── MemberListPage.tsx
│   │   └── settings/
│   │       └── SettingsPage.tsx
│   ├── components/
│   │   ├── Layout/
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Header.tsx
│   │   │   └── Content.tsx
│   │   ├── MarkdownRenderer.tsx     # Markdown 渲染
│   │   ├── FilePreview.tsx          # 文件预览
│   │   ├── LoadingSpinner.tsx
│   │   └── ErrorBoundary.tsx
│   ├── hooks/
│   │   ├── useAuth.ts
│   │   ├── useTasks.ts
│   │   ├── useWebSocket.ts
│   │   └── useDebounce.ts
│   ├── services/
│   │   ├── api.ts                   # Axios 实例
│   │   ├── taskService.ts
│   │   ├── workspaceService.ts
│   │   └── authService.ts
│   ├── store/
│   │   ├── authStore.ts
│   │   ├── taskStore.ts
│   │   └── chatStore.ts
│   ├── types/
│   │   ├── task.ts
│   │   ├── session.ts
│   │   └── api.ts
│   └── utils/
│       ├── date.ts
│       ├── format.ts
│       └── constants.ts
├── public/
├── index.html
├── vite.config.ts
├── tsconfig.json
├── package.json
└── .env.example
```

### 5.2 WebSocket 通信

```typescript
// hooks/useWebSocket.ts

import { useEffect, useRef, useCallback } from 'react';
import { Client } from '@stomp/stompjs';

interface WSMessage {
  type: 'text' | 'tool_call' | 'ask_human' | 'present_result' | 'error';
  content: any;
}

export function useWebSocket(sessionId: string, onMessage: (msg: WSMessage) => void) {
  const clientRef = useRef<Client | null>(null);

  useEffect(() => {
    const client = new Client({
      brokerURL: `wss://${window.location.host}/ws/chat`,
      connectHeaders: {
        Authorization: `Bearer ${localStorage.getItem('token')}`,
      },
      onConnect: () => {
        client.subscribe(`/topic/session/${sessionId}`, (message) => {
          onMessage(JSON.parse(message.body));
        });
      },
    });

    client.activate();
    clientRef.current = client;

    return () => {
      client.deactivate();
    };
  }, [sessionId]);

  const sendMessage = useCallback((content: string) => {
    clientRef.current?.publish({
      destination: `/app/session/${sessionId}/message`,
      body: JSON.stringify({ content }),
    });
  }, [sessionId]);

  return { sendMessage };
}
```

---

## 6. 部署架构

### 6.1 Docker Compose (开发环境)

```yaml
# docker-compose.yml
version: '3.8'

services:
  # API 服务
  elevo-server:
    build: ./elevo-server
    ports:
      - "8080:8080"
    environment:
      - DB_PASSWORD=dev_password
      - JWT_SECRET=dev_jwt_secret
      - REDIS_PASSWORD=dev_redis_password
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    depends_on:
      - postgres
      - redis
      - nats

  # AI 引擎
  elevo-ai:
    build: ./elevo-ai
    ports:
      - "8000:8000"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - MCP_SERVER_URL=http://elevo-server:8080
    depends_on:
      - elevo-server

  # 前端
  elevo-web:
    build: ./elevo-web
    ports:
      - "3000:80"
    depends_on:
      - elevo-server

  # PostgreSQL
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: elevo
      POSTGRES_USER: elevo
      POSTGRES_PASSWORD: dev_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"

  # Redis
  redis:
    image: redis:7-alpine
    command: redis-server --requirepass dev_redis_password
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"

  # NATS
  nats:
    image: nats:2-alpine
    ports:
      - "4222:4222"
      - "8222:8222"

  # MinIO (对象存储)
  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"

volumes:
  postgres_data:
  redis_data:
  minio_data:
```

### 6.2 Kubernetes (生产环境)

```yaml
# deployments/k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: elevo-server
  labels:
    app: elevo-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: elevo-server
  template:
    metadata:
      labels:
        app: elevo-server
    spec:
      containers:
      - name: server
        image: elevo/server:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: "2"
            memory: 1Gi
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: elevo-secrets
              key: db-password
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
```

---

## 7. API 完整路由表

| Method | Path | 说明 |
|---|---|---|
| **认证** | | |
| POST | /api/v1/auth/login | 登录 |
| POST | /api/v1/auth/refresh | 刷新 Token |
| POST | /api/v1/auth/logout | 登出 |
| **工作空间** | | |
| GET | /api/v1/workspaces | 列出用户所属工作空间 |
| POST | /api/v1/workspaces | 创建工作空间 |
| GET | /api/v1/workspaces/:id | 获取工作空间详情 |
| PATCH | /api/v1/workspaces/:id | 更新工作空间 |
| **会话** | | |
| POST | /api/v1/workspaces/:id/sessions | 创建会话 |
| GET | /api/v1/sessions/:id | 获取会话信息 |
| DELETE | /api/v1/sessions/:id | 关闭会话 |
| POST | /api/v1/sessions/:id/message | 发送消息给 AI |
| WS | /ws/chat | WebSocket 连接 |
| **任务** | | |
| POST | /api/v1/workspaces/:id/tasks | 创建任务 |
| GET | /api/v1/workspaces/:id/tasks | 查询任务列表 |
| GET | /api/v1/workspaces/:id/tasks/:taskId | 获取任务详情 |
| PATCH | /api/v1/workspaces/:id/tasks/:taskId | 更新任务 |
| DELETE | /api/v1/workspaces/:id/tasks/:taskId | 删除任务 |
| POST | /api/v1/workspaces/:id/tasks/:taskId/comments | 添加评论 |
| GET | /api/v1/workspaces/:id/tasks/:taskId/comments | 获取评论列表 |
| GET | /api/v1/workspaces/:id/tasks/:taskId/activity | 获取活动日志 |
| **业务对象** | | |
| GET | /api/v1/workspaces/:id/objects | 列出业务对象类型 |
| POST | /api/v1/workspaces/:id/objects | 创建业务对象类型 |
| POST | /api/v1/workspaces/:id/instances | 创建实例 |
| GET | /api/v1/workspaces/:id/instances | 列出实例 |
| GET | /api/v1/workspaces/:id/instances/:instId | 获取实例 |
| PATCH | /api/v1/workspaces/:id/instances/:instId | 更新实例 |
| PATCH | /api/v1/workspaces/:id/instances/:instId/state | 更新状态 |
| **定时任务** | | |
| POST | /api/v1/workspaces/:id/schedules | 创建定时任务 |
| GET | /api/v1/workspaces/:id/schedules | 列出定时任务 |
| GET | /api/v1/workspaces/:id/schedules/:jobId | 获取定时任务 |
| PATCH | /api/v1/workspaces/:id/schedules/:jobId | 更新定时任务 |
| PATCH | /api/v1/workspaces/:id/schedules/:jobId/toggle | 启用/禁用 |
| POST | /api/v1/workspaces/:id/schedules/:jobId/run | 手动执行 |
| DELETE | /api/v1/workspaces/:id/schedules/:jobId | 删除 |
| **成员** | | |
| GET | /api/v1/workspaces/:id/members | 列出成员 |
| GET | /api/v1/workspaces/:id/members/search | 搜索成员 |
| PATCH | /api/v1/workspaces/:id/members/:userId/role | 更新角色 |
| **文件** | | |
| POST | /api/v1/workspaces/:id/files/upload | 上传文件 |
| GET | /api/v1/workspaces/:id/files/:path | 下载/预览文件 |
| GET | /api/v1/files/:fileId/preview | 获取预览 URL |
| **健康检查** | | |
| GET | /healthz | 存活探针 |
| GET | /readyz | 就绪探针 |
| GET | /metrics | Prometheus 指标 |

---

## 8. 错误码规范

```go
// pkg/errors/codes.go

const (
    // 通用错误 1xxx
    ErrInternal       = 1000
    ErrBadRequest     = 1001
    ErrUnauthorized   = 1002
    ErrForbidden      = 1003
    ErrNotFound       = 1004
    ErrConflict       = 1009
    ErrRateLimited    = 1010

    // 认证错误 2xxx
    ErrInvalidToken   = 2001
    ErrTokenExpired   = 2002
    ErrInvalidCredentials = 2003

    // 工作空间错误 3xxx
    ErrWorkspaceNotFound   = 3001
    ErrWorkspaceAccessDenied = 3002

    // 任务错误 4xxx
    ErrTaskNotFound     = 4001
    ErrParentNotFound   = 4002
    ErrInvalidParentType = 4003
    ErrUserNotMember    = 4004

    // 业务对象错误 5xxx
    ErrObjectNotFound   = 5001
    ErrObjectDuplicate  = 5002
    ErrInvalidTransition = 5003

    // 沙箱错误 6xxx
    ErrSandboxCreate    = 6001
    ErrSandboxTimeout   = 6002
    ErrSandboxForbidden = 6003
)
```

---

## 9. 监控指标

### 9.1 Prometheus 指标

| 指标名称 | 类型 | 说明 |
|---|---|---|
| `elevo_http_requests_total` | Counter | HTTP 请求总数 |
| `elevo_http_request_duration_seconds` | Histogram | 请求延迟 |
| `elevo_ai_messages_total` | Counter | AI 消息总数 |
| `elevo_ai_message_duration_seconds` | Histogram | AI 响应延迟 |
| `elevo_ai_tool_calls_total` | Counter | 工具调用总数 |
| `elevo_ai_tool_call_duration_seconds` | Histogram | 工具调用延迟 |
| `elevo_sandbox_containers_active` | Gauge | 活跃沙箱容器数 |
| `elevo_sandbox_container_duration_seconds` | Histogram | 容器生命周期 |
| `elevo_sessions_active` | Gauge | 活跃会话数 |
| `elevo_prompt_cache_hits` | Counter | 提示词缓存命中 |
| `elevo_prompt_cache_misses` | Counter | 提示词缓存未命中 |
