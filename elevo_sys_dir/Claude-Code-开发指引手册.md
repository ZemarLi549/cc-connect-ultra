# Claude Code 开发指引手册

> **本文档专为 Claude Code 编写**，是 AI 辅助开发时的核心执行手册。
> 在使用 Claude Code 进行开发时，请将本文档作为 CLAUDE.md 或项目上下文的一部分注入。

---

## 0. 快速开始

```
你是一个全栈开发工程师，负责开发一个 AI 智能工作平台。
请严格按照本文档、PRD、技术设计文档和 SOW 进行开发。
所有代码必须可编译、可运行、通过测试。
```

---

## 1. 项目结构

```
project-root/
├── elevo-server/          # Go 后端服务
├── elevo-ai/              # Python AI 引擎
├── elevo-web/             # React 前端
├── elevo-sandbox/         # Docker 沙箱镜像
├── migrations/            # SQL 迁移脚本
├── deployments/           # Docker/K8s 配置
├── docs/                  # 项目文档
│   ├── PRD-AI-智能工作平台.md
│   ├── Technical-Design-AI-智能工作平台.md
│   └── SOW-AI-智能工作平台.md
├── CLAUDE.md              # 本文件（Claude Code 指引）
├── docker-compose.yml     # 开发环境
├── Makefile               # 构建命令
└── README.md              # 项目说明
```

---

## 2. 技术栈（严格遵循）

### 后端 (elevo-server)
- **语言**: Go 1.22+
- **框架**: Gin (HTTP), GORM (ORM)
- **包管理**: Go Modules
- **代码风格**: gofmt + go vet

### AI 引擎 (elevo-ai)
- **语言**: Python 3.11+
- **框架**: FastAPI (HTTP), httpx (Async HTTP)
- **LLM SDK**: anthropic (Official Anthropic Python SDK)
- **代码风格**: PEP 8 + Black + isort

### 前端 (elevo-web)
- **框架**: React 18 + TypeScript
- **构建**: Vite 5
- **UI 库**: Ant Design 5
- **状态管理**: Zustand
- **HTTP**: Axios
- **WebSocket**: @stomp/stompjs
- **代码风格**: ESLint + Prettier

### 数据库
- **PostgreSQL 16** (必须使用 JSONB)
- **Redis 7** (缓存)

---

## 3. 开发规范（必须遵守）

### 3.1 Go 代码规范

```go
// ✅ 正确：错误处理必须显式处理
result, err := service.DoSomething(ctx)
if err != nil {
    return nil, fmt.Errorf("service.DoSomething failed: %w", err)
}

// ❌ 错误：忽略错误
result, _ := service.DoSomething(ctx)

// ✅ 正确：使用 context 传递请求上下文
func (h *Handler) GetTask(c *gin.Context) {
    ctx := c.Request.Context()
    workspaceID := c.GetString("workspace_id")
    // ...
}

// ✅ 正确：统一响应格式
c.JSON(http.StatusOK, response.Success(data))
c.JSON(http.StatusBadRequest, response.Error(response.ErrBadRequest, "invalid input"))

// ✅ 正确：GORM 查询
func (r *TaskRepo) List(ctx context.Context, workspaceID string, filter *TaskFilter) ([]*model.Task, int64, error) {
    var tasks []*model.Task
    var total int64

    query := r.db.WithContext(ctx).Where("workspace_id = ?", workspaceID)

    if len(filter.Status) > 0 {
        query = query.Where("status IN ?", filter.Status)
    }

    if err := query.Model(&model.Task{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }

    offset := (filter.Page - 1) * filter.PageSize
    if err := query.Order(filter.SortOrder + " " + filter.SortBy).
        Offset(offset).Limit(filter.PageSize).
        Find(&tasks).Error; err != nil {
        return nil, 0, err
    }

    return tasks, total, nil
}
```

### 3.2 Python 代码规范

```python
# ✅ 正确：使用 async/await
from fastapi import APIRouter, Depends, HTTPException
from anthropic import Anthropic

router = APIRouter()

@router.post("/sessions/{session_id}/message")
async def process_message(
    session_id: str,
    request: MessageRequest,
    session: AsyncSession = Depends(get_db_session),
):
    result = await agent_runtime.process_message(
        session_id=session_id,
        workspace_id=request.workspace_id,
        user_id=request.user_id,
        message=request.content,
    )
    return {"content": result}

# ✅ 正确：使用 Pydantic 模型
from pydantic import BaseModel, Field

class CreateTaskRequest(BaseModel):
    title: str = Field(..., min_length=1, max_length=500)
    description: str = Field(default="", max_length=5000)
    type: Literal["task", "goal", "reminder"] = "task"
    priority: Literal["urgent", "high", "medium", "low"] = "medium"
    due_date: datetime | None = None
    tags: list[str] = Field(default_factory=list)

# ❌ 错误：使用 dict 而非 Pydantic
def create_task(data: dict):  # 不要这样做
    pass
```

### 3.3 React 代码规范

```typescript
// ✅ 正确：使用函数组件 + hooks
import React, { useState, useEffect } from 'react';
import { Button, Card, List, Tag } from 'antd';
import { useTasks } from '@/hooks/useTasks';

interface TaskListPageProps {
  workspaceId: string;
}

const TaskListPage: React.FC<TaskListPageProps> = ({ workspaceId }) => {
  const { tasks, loading, pagination, fetchTasks } = useTasks(workspaceId);
  const [filters, setFilters] = useState<TaskFilters>({});

  useEffect(() => {
    fetchTasks(filters);
  }, [filters, workspaceId]);

  return (
    <div className="task-list-page">
      <TaskFilters value={filters} onChange={setFilters} />
      <List
        loading={loading}
        dataSource={tasks}
        pagination={{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          onChange: (page, pageSize) => {
            fetchTasks({ ...filters, page, page_size: pageSize });
          },
        }}
        renderItem={(task) => (
          <List.Item>
            <TaskCard task={task} />
          </List.Item>
        )}
      />
    </div>
  );
};

// ❌ 错误：使用 class 组件
class TaskListPage extends React.Component { ... }  // 不要这样做

// ❌ 错误：使用 any 类型
const handleClick = (data: any) => { ... }  // 不要这样做
```

### 3.4 命名规范

| 元素 | 规范 | 示例 |
|---|---|---|
| Go 包名 | 小写，无下划线 | `task`, `businessobject` |
| Go 文件名 | snake_case | `task_service.go` |
| Python 文件名 | snake_case | `skill_engine.py` |
| Python 类名 | PascalCase | `SkillEngine`, `AgentRuntime` |
| Python 函数 | snake_case | `process_message()`, `build_prompt()` |
| React 组件 | PascalCase | `TaskListPage`, `MessageBubble` |
| React 文件名 | PascalCase.tsx | `TaskListPage.tsx` |
| React hooks | camelCase, use 前缀 | `useTasks`, `useWebSocket` |
| TypeScript 接口 | PascalCase, I 前缀可选 | `Task`, `TaskFilters` |
| API 路由 | kebab-case | `/api/v1/scheduled-jobs` |
| 数据库表名 | snake_case, 复数 | `tasks`, `business_objects` |
| 数据库列名 | snake_case | `workspace_id`, `due_date` |
| 环境变量 | UPPER_SNAKE_CASE | `DATABASE_HOST`, `JWT_SECRET` |

---

## 4. 关键实现细节（Claude Code 易犯错误）

### 4.1 ❌ 常见错误：workspace_id 未传递

所有数据操作 API 都必须验证 workspace_id。这是最常见的错误。

```go
// ✅ 正确：每个 handler 都检查 workspace_id
func (h *TaskHandler) Create(c *gin.Context) {
    workspaceID, exists := c.Get("workspace_id")
    if !exists {
        c.JSON(http.StatusBadRequest, response.Error(response.ErrBadRequest, "workspace_id is required"))
        return
    }
    // 继续处理...
}
```

### 4.2 ❌ 常见错误：assignee_id 特殊值未处理

```go
// ✅ 正确：处理 me / none / 具体ID 三种情况
switch req.AssigneeID {
case "me":
    task.AssigneeID = &currentUserID
case "none", "":
    task.AssigneeID = nil
default:
    // 验证用户是工作空间成员
    if !s.memberRepo.IsMember(ctx, wsID, req.AssigneeID) {
        return nil, ErrUserNotMember
    }
    task.AssigneeID = &req.AssigneeID
}
```

### 4.3 ❌ 常见错误：tags 是追加而非替换

```go
// ✅ 正确：tags 是完全替换语义
func (s *TaskService) UpdateTags(ctx context.Context, taskID uuid.UUID, newTags []string) error {
    // 先删除所有旧标签
    if err := s.repo.ClearTags(ctx, taskID); err != nil {
        return err
    }
    // 再添加新标签
    for _, tag := range newTags {
        if err := s.repo.AddTag(ctx, taskID, tag); err != nil {
            return err
        }
    }
    return nil
}
```

### 4.4 ❌ 常见错误：is_overdue 用触发器而非计算列

```sql
-- ✅ 正确：使用 GENERATED ALWAYS AS (计算列)
is_overdue BOOLEAN GENERATED ALWAYS AS (
    due_date IS NOT NULL
    AND due_date < NOW()
    AND status NOT IN ('done', 'cancelled')
) STORED

-- ❌ 错误：用触发器或应用层计算
```

### 4.5 ❌ 常见错误：MCP 工具 data 参数接受字符串

```python
# ✅ 正确：data 必须是 dict，不能是 JSON 字符串
if isinstance(request.data, str):
    raise HTTPException(
        status_code=400,
        detail="data must be an object, not a JSON string"
    )
```

### 4.6 ❌ 常见错误：技能触发用 LLM 而非关键词

```python
# ✅ 正确：技能触发使用关键词/正则匹配
def match(self, message: str) -> str | None:
    for skill_name, skill in self.skills.items():
        for trigger in skill.get("triggers", []):
            if trigger.startswith("^"):
                if re.search(trigger, message, re.IGNORECASE):
                    return skill_name
            elif trigger.lower() in message.lower():
                return skill_name
    return None

# ❌ 错误：用 LLM 做意图分类来触发技能（太慢且不稳定）
```

### 4.7 ❌ 常见错误：定时任务执行未创建独立会话

```python
# ✅ 正确：每次执行创建独立的 AI 会话
async def execute_job(job: ScheduledJob):
    session = await create_session(
        workspace_id=job.workspace_id,
        type="scheduled",
    )
    try:
        result = await agent_runtime.process_message(
            session_id=session.id,
            workspace_id=job.workspace_id,
            user_id=job.created_by,
            message=job.prompt,
        )
        job.last_run_status = "success"
    except TimeoutError:
        job.last_run_status = "timeout"
        job.consecutive_failures += 1
    except Exception as e:
        job.last_run_status = "failed"
        job.consecutive_failures += 1
    finally:
        await close_session(session.id)
        job.last_run_at = datetime.utcnow()
        if job.consecutive_failures >= MAX_FAILURES:
            job.enabled = False
            job.pause_reason = f"Auto-paused after {MAX_FAILURES} consecutive failures"
```

### 4.8 ❌ 常见错误：状态机验证被跳过

```python
# ✅ 正确：默认验证状态转换，force=True 才跳过
def transition_state(self, instance: ObjectInstance, new_state: str, force: bool = False):
    if not force:
        machine = self.get_state_machine(instance.object_type)
        if machine and not machine.can_transition(instance.current_state, new_state):
            raise InvalidTransitionError(
                f"Cannot transition from '{instance.current_state}' to '{new_state}'"
            )
    instance.current_state = new_state
```

### 4.9 ❌ 常见错误：AI 回复语言不跟随用户

```python
# ✅ 正确：在系统提示词中强调语言适配规则
LANGUAGE_RULE = """
## Language Adaptation
All visible content MUST use the same language as the user's current input.
- If user writes in Chinese → respond in Chinese (including thinking, summaries, tool result summaries)
- If user writes in English → respond in English
- Do NOT use a default language
- Do NOT deviate from user's current input language due to UI or historical message language
"""
```

### 4.10 ❌ 常见错误：删除任务未级联

```go
// ✅ 正确：数据库层 ON DELETE CASCADE
// 同时在应用层返回级联删除的信息
func (s *TaskService) Delete(ctx context.Context, taskID uuid.UUID) (*DeleteResult, error) {
    // 递归查找所有子任务
    allIDs := s.repo.GetAllDescendantIDs(ctx, taskID)
    allIDs = append(allIDs, taskID)

    // 数据库 CASCADE 会自动删除，这里只是获取被删除的数量
    count := int64(len(allIDs))

    if err := s.repo.Delete(ctx, taskID); err != nil {
        return nil, err
    }

    return &DeleteResult{
        DeletedCount: count,
        DeletedIDs:   allIDs,
    }, nil
}
```

---

## 5. 数据库关键索引（必须创建）

```sql
-- 任务表索引（性能关键）
CREATE INDEX idx_tasks_workspace ON tasks(workspace_id);
CREATE INDEX idx_tasks_status ON tasks(workspace_id, status);
CREATE INDEX idx_tasks_assignee ON tasks(workspace_id, assignee_id);
CREATE INDEX idx_tasks_type ON tasks(workspace_id, type);
CREATE INDEX idx_tasks_priority ON tasks(workspace_id, priority);
CREATE INDEX idx_tasks_due ON tasks(workspace_id, due_date);
CREATE INDEX idx_tasks_parent ON tasks(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_tasks_created_by ON tasks(workspace_id, created_by);
CREATE INDEX idx_tasks_search ON tasks USING GIN(to_tsvector('simple', title || ' ' || COALESCE(description, '')));

-- 对象实例索引
CREATE INDEX idx_oi_workspace ON object_instances(workspace_id);
CREATE INDEX idx_oi_type ON object_instances(workspace_id, object_type);
CREATE INDEX idx_oi_state ON object_instances(workspace_id, current_state);
CREATE INDEX idx_oi_data ON object_instances USING GIN(data jsonb_path_ops);

-- 定时任务索引
CREATE INDEX idx_sj_next_run ON scheduled_jobs(next_run_at) WHERE enabled = true;

-- 审计日志索引
CREATE INDEX idx_audit_time ON audit_logs(created_at);
CREATE INDEX idx_audit_action ON audit_logs(action);
```

---

## 6. API 响应格式（严格遵守）

### 成功响应
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### 分页响应
```json
{
  "code": 0,
  "message": "success",
  "data": [ ... ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

### 错误响应
```json
{
  "code": 1001,
  "message": "workspace_id is required"
}
```

### 任务对象格式
```json
{
  "id": "uuid",
  "task_number": "task01-42",
  "title": "完成项目报告",
  "description": "...",
  "type": {
    "key": "task",
    "name": "待办任务",
    "icon": "✅",
    "color": "#1890ff"
  },
  "status": {
    "name": "todo",
    "color": "#d9d9d9",
    "category": "open"
  },
  "priority": {
    "name": "high",
    "color": "#fa8c16",
    "level": 2
  },
  "assignee": {
    "id": "uuid",
    "name": "张三",
    "avatar": "https://..."
  },
  "parent": null,
  "due_date": "2026-05-15T18:00:00Z",
  "start_date": null,
  "completed_at": null,
  "created_at": "2026-05-09T10:00:00Z",
  "updated_at": "2026-05-09T10:00:00Z",
  "subtask_count": 2,
  "comment_count": 5,
  "is_overdue": false,
  "tags": ["urgent", "Q2"],
  "context": {
    "session_id": null,
    "message_ids": null
  }
}
```

---

## 7. 系统提示词模板（AI Agent）

以下是注入给 AI Agent 的系统提示词结构，Claude Code 在开发 elevo-ai 时需要使用：

```
你是 **[平台名称]** 的智能助手。

## 你的能力

### 任务与流程管理
- 创建、查询、更新和管理任务（支持类型、状态、优先级、指派人、截止日期、标签等属性）
- 支持层级任务结构（父任务/子任务）
- 追踪任务活动和变更记录

### 业务对象操作
- 基于用户定义的模式创建和管理业务对象实例
- 执行状态转换（支持状态机验证）
- 查询和筛选对象实例

## 行为规则

1. **优先使用技能和命令**完成任务
2. **禁止**编写应用代码
3. **禁止**执行 shell 命令（除非插件明确允许）
4. **禁止**访问网络（除非插件明确允许）
5. 超出能力范围时，说明你能帮什么，而不是尝试自己解决

## 语言适配

所有对用户可见的内容，都必须使用与用户当前这次输入相同的语言。

## Workspace Directory Structure

### ⚠️ PATH RULE — MUST USE RELATIVE PATHS

Your current working directory (cwd) = workspace root.
All file paths MUST be relative (no leading `/`).

### First Action: Read context.json

Before calling ANY MCP tools, read `.session/{session_id}/context.json` to get `workspace_id`.

### Access Rules

- **Writable**: `.session/{session_id}/` and its subdirectories
- **Read-only**: `.elevo/`, `CLAUDE.md`, `AGENTS.md`
- **Forbidden**: Other session directories, absolute paths
```

---

## 8. 测试要求

### 8.1 必须编写的测试

| 模块 | 测试类型 | 关键测试用例 |
|---|---|---|
| 任务服务 | 单元测试 | CRUD、过滤查询、层级关系、级联删除、assignee 解析 |
| 业务对象 | 单元测试 | Schema 校验、状态机转换、JSONB 查询 |
| 定时任务 | 单元测试 | 调度触发、失败处理、暂停恢复、next_run 计算 |
| MCP 工具 | 单元测试 | 参数校验、workspace_id 注入、错误处理 |
| 技能引擎 | 单元测试 | 关键词匹配、正则匹配、技能加载 |
| 认证 | 单元测试 | Token 签发/验证/过期/刷新 |
| 多租户 | 集成测试 | 跨工作空间隔离、权限拒绝 |
| AI 对话 | 集成测试 | 工具调用链路、技能触发、多轮对话 |
| 沙箱 | 集成测试 | 容器创建/销毁、文件隔离、网络隔离 |

### 8.2 测试命令

```bash
# Go 后端测试
cd elevo-server && go test ./... -cover -v

# Python AI 引擎测试
cd elevo-ai && pytest --cov=. --cov-report=term-missing -v

# React 前端测试
cd elevo-web && npm test -- --coverage
```

---

## 9. Makefile 命令

```makefile
.PHONY: dev build test clean migrate

# 开发环境启动
dev:
	docker compose up -d postgres redis nats minio
	cd elevo-server && go run cmd/server/main.go &
	cd elevo-ai && uvicorn app.main:app --reload --port 8000 &
	cd elevo-web && npm run dev

# 数据库迁移
migrate:
	migrate -path migrations -database "postgres://elevo:dev_password@localhost:5432/elevo?sslmode=disable" up

# 构建
build:
	cd elevo-server && go build -o bin/server cmd/server/main.go
	cd elevo-ai && docker build -t elevo-ai:latest .
	cd elevo-web && npm run build

# 测试
test:
	cd elevo-server && go test ./... -cover
	cd elevo-ai && pytest --cov=. -v
	cd elevo-web && npm test

# 代码格式化
fmt:
	cd elevo-server && gofmt -w .
	cd elevo-ai && black . && isort .
	cd elevo-web && npx prettier --write "src/**/*.{ts,tsx}"

# 代码检查
lint:
	cd elevo-server && go vet ./...
	cd elevo-ai && ruff check .
	cd elevo-web && npx eslint src/ --ext .ts,.tsx

# 清理
clean:
	docker compose down -v
	rm -rf elevo-server/bin/
	rm -rf elevo-web/dist/
```

---

## 10. 环境变量

### .env.example

```bash
# 数据库
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=elevo
DATABASE_USER=elevo
DATABASE_PASSWORD=dev_password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=dev_redis_password

# JWT
JWT_SECRET=your-jwt-secret-change-in-production
JWT_EXPIRY=24h

# Anthropic
ANTHROPIC_API_KEY=sk-ant-xxxxx

# MinIO / S3
S3_ENDPOINT=http://localhost:9000
S3_BUCKET=elevo-files
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin

# NATS
NATS_URL=nats://localhost:4222

# 应用
APP_ENV=development
APP_PORT=8080
AI_PORT=8000
```

---

## 11. 开发顺序（严格遵循）

请按以下顺序开发，每个步骤完成后验证再进入下一步：

### Phase 1: 基础 (先完成这些再继续)

1. 初始化项目结构和 Git 仓库
2. 编写全部 SQL 迁移脚本
3. 编写 docker-compose.yml
4. 实现统一响应格式和错误码
5. 实现 JWT 认证（login/refresh/logout）
6. 实现多租户中间件
7. 实现工作空间 CRUD API
8. 实现成员管理 API
9. 编写 Phase 1 测试

### Phase 2: AI 引擎

10. 实现 MCP 协议框架（Client + Server）
11. 实现 Read/Write 文件 MCP 工具
12. 集成 Claude API（tool_use）
13. 实现系统提示词构建器
14. 实现提示词缓存
15. 实现会话管理（context.json 生成）
16. 实现 WebSocket 通信
17. 实现技能引擎
18. 实现 ask_human / present_result
19. 实现 9 个任务管理技能
20. 编写 Phase 2 测试

### Phase 3: 业务功能

21. 实现任务 CRUD MCP 工具
22. 实现任务查询过滤
23. 实现层级任务和级联删除
24. 实现标签和评论系统
25. 实现业务对象定义 API
26. 实现对象实例 CRUD MCP 工具
27. 实现状态机引擎
28. 实现 JSONB 高级查询
29. 实现定时任务调度器
30. 编写 Phase 3 测试

### Phase 4: 前端

31. 实现布局框架
32. 实现聊天页面 + WebSocket
33. 实现消息渲染（Markdown/代码/图片）
34. 实现交互卡片（choice/form/present_result）
35. 实现任务列表页
36. 实现任务详情页
37. 实现工作台 Dashboard
38. 实现飞书 Bot 接入
39. 实现钉钉 Bot 接入
40. 编写 Phase 4 测试

### Phase 5: 安全与部署

41. 构建沙箱 Docker 镜像
42. 实现沙箱管理器
43. 配置 Seccomp + Cgroups
44. 实现网络隔离
45. 实现危险命令检测
46. 实现审计日志
47. 配置 Prometheus 监控
48. 编写 K8s 部署清单
49. 编写 CI/CD Pipeline
50. 编写 API 文档和部署指南
51. 最终端到端验收测试

---

## 12. 常见问题 FAQ

### Q: MCP 协议是自己实现还是用官方 SDK？
A: 参考 Anthropic 官方 MCP SDK (TypeScript/Python) 的协议规范，在 Go 端实现 MCP Server，Python 端实现 MCP Client。MCP 是 JSON-RPC 2.0 协议，通信可以用 HTTP 或 stdio。

### Q: 沙箱容器怎么和工作空间文件系统关联？
A: 通过 Docker Volume 挂载：
- workspace root → `/home/sandbox/project` (read-only)
- `.session/{id}/` → `/home/sandbox/project/.session/{id}/` (read-write)
- `.elevo/` 随 workspace root 一起挂载，已包含在 read-only 中

### Q: 如何实现提示词缓存？
A: 使用 Claude API 的 prompt caching 功能。将系统提示词标记为 cache_control，设置 TTL 5 分钟。在 Redis 中维护缓存 key = hash(system_prompt)，缓存命中时复用。

### Q: 定时任务调度用什么实现？
A: 使用 Python 的 APScheduler 或自研调度器。调度器需要持久化到数据库，服务重启后从数据库恢复调度状态。每分钟扫描一次 `enabled=true AND next_run_at <= NOW()` 的任务。

### Q: WebSocket 怎么和 HTTP API 共存？
A: 使用同一个端口。FastAPI 的 WebSocket 和 HTTP 可以共存。Gin 端用 gorilla/websocket。

### Q: 前端怎么渲染 ask_human 的交互卡片？
A: AI 的回复中如果包含 ask_human 类型的 tool_use，前端解析 tool_use 的参数，渲染对应的 choice 或 form 组件。用户提交后，将结果作为 tool_result 发送回后端。

### Q: 如何处理 LLM 工具调用的循环？
A: 设置最大循环次数（如 10 次）。每次 tool_use 后检查 stop_reason，如果不是 end_turn 则继续。超过最大次数强制停止并返回当前结果。

---

## 13. 安全检查清单（开发完成后验证）

- [ ] 所有 API 路由都有认证中间件保护
- [ ] workspace_id 在每个请求中都被验证
- [ ] 跨工作空间数据访问被正确拒绝
- [ ] SQL 查询使用参数化（防止 SQL 注入）
- [ ] 用户输入被正确校验和清洗
- [ ] JWT Token 过期后正确拒绝
- [ ] 文件上传有类型和大小限制
- [ ] 沙箱容器无法访问宿主机网络
- [ ] 危险 shell 命令被拦截
- [ ] 敏感信息不出现在日志中
- [ ] 密码使用 bcrypt 存储（如需要）
- [ ] CORS 配置正确
- [ ] 速率限制生效
- [ ] 审计日志记录安全相关事件

---

## 14. 性能检查清单

- [ ] 数据库查询使用了正确的索引
- [ ] N+1 查询问题已消除（使用 GORM Preload）
- [ ] Redis 缓存了热点数据
- [ ] 数据库连接池大小合理（默认 50）
- [ ] HTTP Keep-Alive 已启用
- [ ] WebSocket 心跳机制正常
- [ ] 大量并发时的资源限制正确
- [ ] 日志异步写入（不阻塞主流程）
- [ ] 文件操作使用流式处理（大文件不全部加载到内存）
- [ ] 分页查询都有 LIMIT 限制
