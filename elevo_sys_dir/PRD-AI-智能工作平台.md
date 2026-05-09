# 产品需求文档 (PRD) — AI 智能工作平台

> **版本**: v1.0
> **日期**: 2026-05-09
> **状态**: 待评审

---

## 1. 产品概述

### 1.1 产品愿景

构建一个通用 AI 智能工作平台（类似 Elevo），作为企业智能化转型的数字工位。平台通过自然语言交互，将 AI 能力工程化地交付到用户日常工作场景中，支撑任务管理、业务对象管理、企业系统集成、运维自动化等多样化业务场景。

### 1.2 核心定位

- **不是垂直行业应用**，而是通用智能工作平台
- **不是聊天机器人**，而是具备工具调用能力的 AI Agent 工作台
- **不是单一功能产品**，而是可扩展的插件化平台

### 1.3 目标用户

| 用户角色 | 典型场景 |
|---|---|
| 企业管理者 | 工作台概览、任务追踪、目标管理、团队协作 |
| 运维工程师 | CMDB 查询、监控告警、故障排查、自动化运维 |
| 项目经理 | 任务分配、进度跟踪、定时报告、知识沉淀 |
| 普通员工 | 待办管理、日程提醒、信息查询、文件管理 |

### 1.4 核心价值主张

1. **自然语言驱动** — 用说话代替操作，降低工具使用门槛
2. **AI-Native** — AI 不是附加功能，而是核心交互方式
3. **插件化扩展** — 核心平台稳定，能力通过插件按需加载
4. **企业级安全** — 多租户隔离、权限控制、操作审计

---

## 2. 功能需求

### 2.1 工作空间管理 (Workspace)

#### 2.1.1 工作空间生命周期

**描述**: 工作空间是平台的基本隔离单元，每个租户/团队拥有独立工作空间。

| 功能 | 需求详情 |
|---|---|
| 创建工作空间 | 管理员创建，指定名称、描述、所属租户 |
| 配置清单 | 自动生成 `manifest.yaml`，���录 workspace_id、tenant_id、创建时间、已安装插件列表 |
| 数据隔离 | 工作空间间数据完全隔离，用户只能访问所属工作空间的数据 |
| 文件系统 | 每个工作空间拥有独立文件目录结构 |

#### 2.1.2 目录结构规范

```
workspace_root/
├── .elevo/                    # 系统元数据（只读）
│   ├── manifest.yaml          # 工作空间清单
│   ├── sync_state.json        # 同步状态
│   └── sandbox/               # 沙箱运行时
├── CLAUDE.md                  # AI Agent 行为指令（只读）
├── AGENTS.md                  # 技能/Agent 定义（只读）
├── knowledge/                 # 工作空间知识库
├── .session/{session_id}/     # 会话临时目录（可写）
│   ├── context.json           # 会话上下文
│   ├── context/               # @mention 上下文
│   └── documents/             # 上传文件
└── *.md / *.json / ...        # 用户持久文件
```

#### 2.1.3 验收标准

- [ ] 工作空间创建后自动生成 manifest.yaml
- [ ] 不同工作空间的用户无法互相访问数据
- [ ] 系统目录（.elevo/）对 AI Agent 不可写
- [ ] 会话目录在会话结束后自动清理
- [ ] 工作空间支持插件安装/卸载

---

### 2.2 会话管理 (Session)

#### 2.2.1 会话生命周期

**描述**: 每次用户对话创建一个独立会话，拥有完整的上下文和隔离环境。

| 状态 | 说明 |
|---|---|
| created | 会话创建，生成 session_id |
| active | 用户与 AI 正在对话 |
| idle | 超过 30 分钟无操作 |
| closed | 用户主动关闭或系统超时关闭 |

#### 2.2.2 会话上下文

**context.json 结构**:

```json
{
  "workspace_id": "uuid-v4",
  "user_id": "uuid-v4",
  "conversation_id": "uuid-v4 (same as session_id)",
  "created_at": "ISO8601 timestamp",
  "status": "active"
}
```

**要求**:
- 会话启动时自动创建 context.json
- AI Agent 每次响应前必须读取 context.json 获取 workspace_id
- workspace_id 是所有数据操作的必要参数，缺失则工具调用失败

#### 2.2.3 沙箱执行环境

**要求**:
- 每个会话运行在独立的 Docker 容器中
- 容器内工作目录为 workspace root 的挂载
- 系统目录（.elevo/）以只读方式挂载
- 会话目录（.session/{id}/）以读写方式挂载
- 其他会话目录不可访问
- 容器资源限制：CPU 2 核、内存 4GB、磁盘 10GB
- 容器内禁止网络访问（除非插件明确允许）

#### 2.2.4 验收标准

- [ ] 新会话自动创建沙箱容器
- [ ] context.json 正确生成且包含必要字段
- [ ] AI 无法写入 .elevo/ 目录
- [ ] AI 无法访问其他会话目录
- [ ] 会话关闭后容器自动销毁
- [ ] 会话临时文件在关闭后清理

---

### 2.3 AI Agent 引擎

#### 2.3.1 对话模型集成

**要求**:
- 支持 Claude (Anthropic) 作为主要 LLM 后端
- 通过 Claude Agent SDK 实现工具调用
- 支持 System Prompt 注入（包含规则、技能、工具定义）
- 支持提示词缓存（默认 TTL 5 分钟，可配置）

**System Prompt 结构**:

```
[角色定义] — Agent 身份和行为规范
[工具定义] — MCP 工具列表及参数 Schema
[技能列表] — 可用 Skills 及触发条件
[行为规则] — 禁止事项、安全规则
[工作空间配置] — 目录结构、路径规则、访问权限
[会话上下文] — workspace_id, user_id 等
```

#### 2.3.2 工具调用机制

**MCP (Model Context Protocol) 集成**:

| 组件 | 说明 |
|---|---|
| MCP Client | 内置于 AI Agent，发送工具调用请求 |
| MCP Servers | 各能力提供方（任务管理、文件操作、企业集成等） |
| 工具注册 | 每个工具通过 JSON Schema 定义名称、描述、参数 |
| 工具调用流程 | 用户输入 → LLM 决策 → MCP Client → MCP Server → 返回结果 → LLM 生成回复 |

**工具分类**:

| 类别 | 工具举例 |
|---|---|
| 平台工具 | ask_human、present_result |
| 任务管理 | create_task、list_tasks、update_task、delete_task |
| 业务对象 | create_object_instance、list_object_instances |
| 定时任务 | create_scheduled_job、list_scheduled_jobs |
| 文件操作 | Read、Write、Edit |
| 企业集成 | IM 消息、邮件、日历、SAP 等（通过插件） |

#### 2.3.3 意图识别与技能调度

**技能触发流程**:

```
用户输入 → 关键词/正则匹配 → 匹配成功? → 加载技能详细说明 → 按说明执行
                                          → 匹配失败 → 直接对话回复或调用 MCP 工具
```

**预置技能列表**:

| 技能名称 | 触发关键词 |
|---|---|
| create-task | 创建任务、新建任务、添加待办、帮我记一下、提醒我、创建目标、新建提醒 |
| list-tasks | 查看任务、我的任务、任务列表、待办事项、今天的任务、逾期任务 |
| update-task | 更新任务、完成任务、标记完成、修改任务、改优先级 |
| delete-task | 删除任务、移除任务、把任务删掉 |
| get-task | 查看任务详情、show task details |
| subtask | 添加子任务、创建子任务、子任务、分解任务 |
| batch-update | 批量更新、把所有...任务、mark all as done |
| dashboard | 任务统计、工作概览、今日工作、周报数据 |
| scheduler | 定时任务、每天...点、每周...、设置提醒、cron job |

#### 2.3.4 人机协作 (ask_human)

**描述**: 当 AI 需要用户确认决策或收集结构化信息时，通过 ask_human 工具发起交互。

**支持的交互类型**:

**1. 选择题 (choice)**:
```json
{
  "questions": [{
    "type": "choice",
    "question": "请选择任务优先级",
    "header": "优先级",
    "multiSelect": false,
    "options": [
      {"label": "紧急", "description": "需要立即处理"},
      {"label": "高", "description": "今日内完成"},
      {"label": "中", "description": "本周内完成"},
      {"label": "低", "description": "可延后处理"}
    ]
  }]
}
```

**2. 表单 (form)**:
```json
{
  "questions": [{
    "type": "form",
    "question": "创建新任务",
    "header": "任务信息",
    "fields": [
      {"name": "title", "label": "任务标题", "type": "text", "required": true},
      {"name": "description", "label": "描述", "type": "textarea"},
      {"name": "priority", "label": "优先级", "type": "select", "options": ["urgent", "high", "medium", "low"]},
      {"name": "due_date", "label": "截止日期", "type": "text"}
    ]
  }]
}
```

**约束**:
- 一次最多 4 个问题
- choice 模式 2-4 个选项
- 调用后 AI 必须停止等待用户响应
- 不能用纯文本列出选项再调用 ask_human（必须直接调用）

#### 2.3.5 结果展示 (present_result)

**描述**: 成功创建或更新工作空间资源后，渲染结构化卡片。

```json
{
  "title": "任务创建成功",
  "summary": "已为您创建 2 个任务",
  "items": [
    {
      "type": "resource",
      "resource_type": "task",
      "resource_id": "uuid",
      "title": "完成项目报告",
      "description": "截止日期: 2026-05-15"
    }
  ]
}
```

**规则**:
- 支持 1-10 个 items
- resource_type: task / goal / reminder / file
- 列表超过 10 个时用纯文本总结
- 不输出原始 URL 或 workspace_id

#### 2.3.6 语言适配

**规则**: AI 回复语言必须与用户当前输入语言一致。
- 用户输入中文 → 所有回复、思考、工具结果总结用中文
- 用户输入英文 → 所有内容用英文
- 不因历史消息或界面语言偏离

#### 2.3.7 验收标准

- [ ] 系统提示词正确注入且包含所有必要部分
- [ ] MCP 工具调用成功率达 99%+
- [ ] 技能关键词匹配准确率 > 95%
- [ ] ask_human 交互卡片正确渲染
- [ ] present_result 资源卡片可点击跳转
- [ ] 语言适配正确切换
- [ ] 提示词缓存命中率 > 80%

---

### 2.4 任务管理系统

#### 2.4.1 数据模型

```sql
CREATE TABLE tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id),
    task_number     VARCHAR(50),           -- 任务编号 (如 "task01-42")
    title           VARCHAR(500) NOT NULL,
    description     TEXT,
    type            VARCHAR(20) DEFAULT 'task',  -- task / goal / reminder
    status          VARCHAR(20) DEFAULT 'todo',  -- todo / in_progress / done / cancelled
    priority        VARCHAR(10) DEFAULT 'medium', -- urgent / high / medium / low / none
    assignee_id     UUID REFERENCES users(id),
    created_by      UUID REFERENCES users(id),
    parent_id       UUID REFERENCES tasks(id),   -- 父任务 ID
    due_date        TIMESTAMPTZ,
    start_date      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    is_overdue      BOOLEAN GENERATED ALWAYS AS (
        due_date < NOW() AND status NOT IN ('done', 'cancelled')
    ) STORED
);
```

```sql
CREATE TABLE task_tags (
    task_id      UUID REFERENCES tasks(id) ON DELETE CASCADE,
    tag_name     VARCHAR(100) NOT NULL,
    PRIMARY KEY (task_id, tag_name)
);
```

```sql
CREATE TABLE task_comments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id      UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id),
    content      TEXT NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
```

#### 2.4.2 任务类型

| 类型 | 图标 | 颜色 | 说明 |
|---|---|---|---|
| task | ✅ | 蓝色 | 普通待办任务 |
| goal | 🎯 | 紫色 | 目标/里程碑，支持 OKR 层级 |
| reminder | 🔔 | 黄色 | 提醒事项 |

#### 2.4.3 任务状态流转

```
                    ┌──────────┐
                    │   todo   │
                    └────┬─────┘
                         │ 开始
                    ┌────▼─────┐
              ┌────►│in_progress│
              │     └────┬─────┘
              │          │ 完成
              │     ┌────▼─────┐
              │     │   done   │
              │     └──────────┘
              │
         ┌────┴─────┐
         │cancelled │
         └──────────┘
```

**状态枚举**:

| 状态 | 分类 | 说明 |
|---|---|---|
| todo | open | 待办 |
| in_progress | open | 进行中 |
| done | closed | 已完成 |
| cancelled | closed | 已取消 |

#### 2.4.4 优先级定义

| 级别 | level | 颜色 |
|---|---|---|
| urgent | 1 | 红色 |
| high | 2 | 橙色 |
| medium | 3 | 蓝色 |
| low | 4 | 灰色 |
| none | 5 | 默认 |

#### 2.4.5 核心 API

**创建任务**:

```yaml
POST /api/v1/workspaces/{workspace_id}/tasks
Body (单个):
  title: string (必填)
  description: string
  type: "task" | "goal" | "reminder" (默认 task)
  priority: "urgent" | "high" | "medium" | "low" (默认 medium)
  assignee_id: string | "me" | "none"
  due_date: ISO8601
  start_date: ISO8601
  parent_id: string (父任务 ID)
  tags: string[]

Body (批量, 最多 10 个):
  tasks: Array<{title, description, type, priority, assignee_id, due_date, tags}>

Response:
  task: TaskObject | { created: TaskObject[], failed: FailedTask[] }
```

**查询任务列表**:

```yaml
GET /api/v1/workspaces/{workspace_id}/tasks
Query Parameters:
  status: string | string[]        # todo, in_progress, done, cancelled
  type: string | string[]          # task, goal, reminder
  priority: string | string[]      # urgent, high, medium, low
  assignee_id: string              # "me" | "unassigned" | 具体用户 ID
  created_by: string               # "me"
  due_date: object                 # 见下方
  tags: object                     # { include: string[], match: "any" | "all" }
  parent_id: string | "null"       # "null" = 仅顶级任务
  keyword: string                  # 标题和描述模糊搜索
  sort_by: "due_date" | "priority" | "created_at" | "updated_at" | "title"
  sort_order: "asc" | "desc"
  page: integer (默认 1)
  page_size: integer (默认 20, 最大 100)

due_date 过滤:
  { today: true }                  # 今天截止
  { this_week: true }              # 本周截止
  { overdue: true }                # 已逾期
  { before: "ISO8601" }            # 在此日期之前
  { after: "ISO8601" }             # 在此日期之后
  { is_null: true }                # 无截止时间

Response:
  items: TaskObject[]
  total: integer
  page: integer
  page_size: integer
  total_pages: integer
```

**获取单个任务**:

```yaml
GET /api/v1/workspaces/{workspace_id}/tasks/{id}
GET /api/v1/workspaces/{workspace_id}/tasks?task_number={number}

# task_number 支持格式: "48", "task01-48", "ENG-42", "#48"
```

**更新任务**:

```yaml
PATCH /api/v1/workspaces/{workspace_id}/tasks/{id}
Body (部分更新，仅传需要修改的字段):
  title: string
  description: string
  status: "todo" | "in_progress" | "done" | "cancelled"
  type: "task" | "goal" | "reminder"
  priority: "urgent" | "high" | "medium" | "low" | "none"
  assignee_id: string | "me" | "none"
  due_date: ISO8601 | "none"
  start_date: ISO8601 | "none"
  parent_id: string | "none"
  tags: string[] (完全替换)

Response:
  task: TaskObject
  changes: Array<{field: string, from: any, to: any}>
```

**删除任务**:

```yaml
DELETE /api/v1/workspaces/{workspace_id}/tasks/{id}

# 级联删除所有子任务、评论、标签
Response:
  deleted_count: integer
  deleted_ids: string[]
```

#### 2.4.6 任务对象完整结构

```typescript
interface TaskObject {
  // 基本信息
  id: string;
  task_number: string;
  title: string;
  description: string;

  // 类型与状态
  type: {
    key: string;       // task / goal / reminder
    name: string;      // 显示名
    icon: string;      // 图标
    color: string;     // 颜色
  };
  status: {
    name: string;      // todo / in_progress / done / cancelled
    color: string;
    category: "open" | "closed";
  };
  priority: {
    name: string;
    color: string;
    level: number;     // 1-5
  };

  // 关联
  assignee: {
    id: string;
    name: string;
    avatar: string;    // 头像 URL
  } | null;
  created_by: {
    id: string;
    name: string;
  };
  parent: {
    id: string;
    title: string;
  } | null;

  // 时间
  start_date: string | null;     // ISO8601
  due_date: string | null;       // ISO8601
  completed_at: string | null;
  created_at: string;
  updated_at: string;

  // 统计
  subtask_count: number;
  comment_count: number;
  is_overdue: boolean;

  // 标签
  tags: string[];

  // 上下文
  context: {
    session_id: string | null;
    message_ids: string[] | null;
  };
}
```

#### 2.4.7 MCP 工具定义

```json
{
  "name": "create_task",
  "description": "创建任务、目标(goal)、提醒(reminder)",
  "parameters": {
    "workspace_id": {"type": "string", "required": true},
    "title": {"type": "string", "description": "任务标题（单个创建时必填）"},
    "tasks": {"type": "array", "description": "批量创建（最多10个）", "maxItems": 10},
    "description": {"type": "string"},
    "type": {"type": "string", "enum": ["task", "goal", "reminder"], "default": "task"},
    "priority": {"type": "string", "enum": ["urgent", "high", "medium", "low"]},
    "assignee_id": {"type": "string", "description": "指派人，'me'=当前用户"},
    "due_date": {"type": "string", "format": "ISO8601"},
    "parent_id": {"type": "string"},
    "tags": {"type": "array", "items": {"type": "string"}}
  }
}
```

#### 2.4.8 验收标准

- [ ] 任务 CRUD 完整实现
- [ ] 批量创建最多支持 10 个
- [ ] 层级任务结构正常工作（父子关系）
- [ ] 删除任务级联删除子任务
- [ ] assignee_id='me' 正确解析为当前用户
- [ ] assignee_id='none' 正确清除指派
- [ ] task_number 多种格式均能解析
- [ ] due_date 各过滤条件正确
- [ ] is_overdue 计算字段准确
- [ ] tags 完全替换语义（非追加）
- [ ] 任务更新返回变更列表

---

### 2.5 业务对象管理

#### 2.5.1 概述

业务对象引擎是一个**通用的、Schema-less 的数据管理框架**，允许用户定义自己的业务对象模型（类似低代码平台的自定义表单）。

#### 2.5.2 业务对象定义 (Schema)

```sql
CREATE TABLE business_objects (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    name         VARCHAR(200) NOT NULL,       -- 对象类型名称
    description  TEXT,
    schema       JSONB NOT NULL DEFAULT '{}', -- 字段定义
    state_machine JSONB DEFAULT NULL,         -- 状态机定义
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (workspace_id, name)
);
```

**Schema 定义示例**:

```json
{
  "fields": [
    {
      "name": "customer_name",
      "label": "客户名称",
      "type": "string",
      "required": true,
      "max_length": 200
    },
    {
      "name": "amount",
      "label": "金额",
      "type": "number",
      "required": true,
      "min": 0
    },
    {
      "name": "status",
      "label": "状态",
      "type": "string",
      "enum": ["pending", "approved", "rejected"]
    },
    {
      "name": "tags",
      "label": "标签",
      "type": "array",
      "items": {"type": "string"}
    }
  ]
}
```

**状态机定义示例**:

```json
{
  "states": ["draft", "pending_review", "approved", "rejected"],
  "initial_state": "draft",
  "transitions": [
    {"from": "draft", "to": "pending_review"},
    {"from": "pending_review", "to": "approved"},
    {"from": "pending_review", "to": "rejected"}
  ]
}
```

#### 2.5.3 对象实例管理

```sql
CREATE TABLE object_instances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id),
    object_type     VARCHAR(200) NOT NULL,     -- 引用 business_objects.name
    data            JSONB NOT NULL DEFAULT '{}', -- 实例数据
    current_state   VARCHAR(100),              -- 当前状态
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
```

#### 2.5.4 核心 API

**列出业务对象**:

```yaml
GET /api/v1/workspaces/{workspace_id}/business-objects
Query:
  search: string    # 按名称或描述模糊搜索
  page: integer
  page_size: integer (最大 100)

Response:
  items: BusinessObject[]
  total: integer
```

**创建对象实例**:

```yaml
POST /api/v1/workspaces/{workspace_id}/object-instances
Body:
  object_type: string (必填)
  data: object (必填, 必须是对象不能是 JSON 字符串)
  current_state: string (可选, 默认使用状态机的 initial_state)
```

**列出对象实例**:

```yaml
GET /api/v1/workspaces/{workspace_id}/object-instances
Query:
  object_type: string
  current_state: string
  filters: {
    field: string,
    operator: "eq" | "ne" | "gt" | "gte" | "lt" | "lte" | "in" | "nin" | "contains",
    value: any
  }
  sort_by: "created_at" | "updated_at"
  sort_order: "asc" | "desc"
  page: integer
  page_size: integer (最大 100)
```

**更新对象实例**:

```yaml
PATCH /api/v1/workspaces/{workspace_id}/object-instances/{instance_id}
Body:
  data: object (部分更新，仅传需要修改的字段)
  target_state: string (可选, 同时���换状态)
  force: boolean (默认 false, 是否跳过状态机验证)
```

**更新状态**:

```yaml
PATCH /api/v1/workspaces/{workspace_id}/object-instances/{instance_id}/state
Body:
  new_state: string (必填)
  force: boolean (默认 false)
```

#### 2.5.5 验收标准

- [ ] 业务对象定义支持动态 Schema
- [ ] 状态机验证正确执行
- [ ] force=true 跳过状态机验证
- [ ] JSONB 过滤查询支持所有操作符
- [ ] data 参数必须为对象，JSON 字符串应返回错误
- [ ] 不存在的 object_type 返回空列表而非错误

---

### 2.6 定时任务调度

#### 2.6.1 数据模型

```sql
CREATE TABLE scheduled_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID NOT NULL REFERENCES workspaces(id),
    name            VARCHAR(100) NOT NULL,
    prompt          VARCHAR(5000) NOT NULL,    -- AI 执行的提示词
    schedule_type   VARCHAR(10) NOT NULL,      -- once / daily / weekly / monthly
    schedule_config JSONB NOT NULL,            -- 调度配置
    timeout_seconds INTEGER DEFAULT 300,       -- 超时时间 (60-1800)
    enabled         BOOLEAN DEFAULT true,
    next_run_at     TIMESTAMPTZ,
    last_run_at     TIMESTAMPTZ,
    last_run_status VARCHAR(20),              -- success / failed / timeout / cancelled
    consecutive_failures INTEGER DEFAULT 0,
    pause_reason    TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);
```

#### 2.6.2 调度类型

| 类型 | schedule_config | 说明 |
|---|---|---|
| once | `{ at: "ISO8601" }` | 一次性任务 |
| daily | `{ time: "HH:MM", timezone: "Asia/Shanghai" }` | 每天执行 |
| weekly | `{ time: "HH:MM", weekday: 0-6, timezone: "..." }` | 每周执行 |
| monthly | `{ time: "HH:MM", day: 1-31, timezone: "..." }` | 每月执行 |

#### 2.6.3 执行机制

- 调度引擎定期扫描 `enabled=true` 且 `next_run_at <= NOW()` 的任务
- 执行时创建独立的 AI 会话，传入 prompt
- 执行超时后标记为 timeout
- 连续失败达到阈值后自动暂停，记录 pause_reason
- 重新启用时重置 consecutive_failures

#### 2.6.4 API

```yaml
# 创建
POST /api/v1/workspaces/{workspace_id}/scheduled-jobs
Body:
  name: string (最长100字符)
  prompt: string (最长5000字符)
  schedule_type: "once" | "daily" | "weekly" | "monthly"
  schedule_config: object
  timeout_seconds: integer (60-1800, 默认300)

# 列表
GET /api/v1/workspaces/{workspace_id}/scheduled-jobs
Query:
  enabled: boolean
  keyword: string
  page: integer
  page_size: integer

# 更新
PATCH /api/v1/workspaces/{workspace_id}/scheduled-jobs/{job_id}
Body: (部分更新)
  name: string
  prompt: string
  schedule_type: string
  schedule_config: object
  timeout_seconds: integer

# 启用/禁用
PATCH /api/v1/workspaces/{workspace_id}/scheduled-jobs/{job_id}/toggle
Body:
  enabled: boolean

# 手动执行
POST /api/v1/workspaces/{workspace_id}/scheduled-jobs/{job_id}/run

# 删除
DELETE /api/v1/workspaces/{workspace_id}/scheduled-jobs/{job_id}
```

#### 2.6.5 验收标准

- [ ] 一次性任务在指定时间触发后自动删除
- [ ] 周期任务按配置正确触发
- [ ] 修改调度配置后自动重新计算 next_run_at
- [ ] 连续失败自动暂停
- [ ] 手动执行不影响正常调度时间
- [ ] 禁用后暂停执行，重新启用后恢复
- [ ] 时区正确处理

---

### 2.7 文件管理

#### 2.7.1 文件操作

| 操作 | 权限 | 说明 |
|---|---|---|
| 读取工作空间文件 | 只读 | CLAUDE.md, AGENTS.md, 用户文件 |
| 读取会话文件 | 读写 | .session/{id}/ 下的文件 |
| 写入会话文件 | 读写 | 仅限当前会话目录 |
| 读取系统文件 | 只读 | .elevo/ 目录 |
| 读取图片 | 支持 | PNG, JPG |
| 读取 PDF | 支持 | 单次最多 20 页 |

#### 2.7.2 文件路径规则

- **所有路径必须使用相对路径**（从 workspace root 开始）
- **禁止使用绝对路径**（`/opt/...`, `/workspace/...` 等都会失败）
- 上下文文件：`.session/{conversation_id}/context.json`

#### 2.7.3 验收标准

- [ ] 相对路径正常工作
- [ ] 绝对路径返回错误
- [ ] 跨会话目录访问被拒绝
- [ ] .elevo/ 目录写入被拒绝
- [ ] 图片文件可正常读取和展示
- [ ] PDF 分页读取正常

---

### 2.8 用户与权限管理

#### 2.8.1 用户模型

```sql
CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    email        VARCHAR(255),
    name         VARCHAR(200) NOT NULL,
    avatar       VARCHAR(500),
    role         VARCHAR(50) DEFAULT 'member',  -- admin / member / viewer
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
```

#### 2.8.2 工作空间成员

```sql
CREATE TABLE workspace_members (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    user_id      UUID NOT NULL REFERENCES users(id),
    role         VARCHAR(50) DEFAULT 'member',
    joined_at    TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (workspace_id, user_id)
);
```

#### 2.8.3 成员搜索

```yaml
GET /api/v1/workspaces/{workspace_id}/members
Query:
  search: string    # 按名称或邮箱模糊匹配
  page: integer
  page_size: integer (最大 100)

Response:
  items: [{
    id: string,
    user_id: string,
    name: string,
    email: string,
    role: string
  }]
```

---

### 2.9 IM 集成 (飞书/钉钉/Slack)

#### 2.9.1 功能要求

| 功能 | 说明 |
|---|---|
| 群聊消息查询 | 获取 IM 群聊的历史消息记录 |
| 消息发送 | 通过 Bot 向群聊或个人发送消息 |
| @机器人触发 | 群聊中 @机器人 触发 AI 对话 |
| 上下文关联 | 消息可关联到任务上下文 |

#### 2.9.2 群聊历史消息 API

```yaml
GET /api/v1/chat-history
Query:
  chat_id: string (必填, 群聊 ID)
  hours: integer (默认24, 最大72)
  limit: integer (默认50, 最大200)

Response:
  items: [{
    sender_name: string,
    content: string,
    message_type: "text" | "image" | "file" | ...,
    created_at: ISO8601
  }]
  # 按时间从旧到新排列
```

---

### 2.10 通知与安全

#### 2.10.1 安全要求

| 要求 | 说明 |
|---|---|
| 沙箱隔离 | 每个会话独立容器 |
| 系统文件保护 | .elevo/ 不可写 |
| 操作审计 | 记录危险操作尝试 |
| 身份验证 | JWT Token + API Key |
| 数据隔离 | workspace_id 级别逻辑隔离 |
| 命令管控 | 沙箱内禁止危险 shell 命令 |
| 输入校验 | 所有 API 参数类型和范围校验 |
| 敏感数据过滤 | 不输出 workspace_id、原始 URL 等 |

---

## 3. 非功能需求

### 3.1 性能

| 指标 | 要求 |
|---|---|
| API 响应时间 (P95) | < 500ms（不含 LLM 调用） |
| AI 首字响应时间 (P95) | < 3s |
| MCP 工具调用延迟 | < 200ms |
| 并发会话数 | ≥ 100 |
| 单工作空间任务数 | ≥ 100,000 |

### 3.2 可用性

| 指标 | 要求 |
|---|---|
| 系统可用性 | ≥ 99.9% |
| 计划内维护 | 滚动部署，零停机 |
| 故障恢复 | < 5 分钟（自动） |

### 3.3 安全

| 指标 | 要求 |
|---|---|
| 数据加密 | TLS 1.3 传输，AES-256 存储 |
| 认证方式 | JWT + 可选 SSO |
| 密码策略 | bcrypt, ≥ 12 轮 |
| 审计日志 | 保留 90 天 |
| 沙箱逃逸防护 | seccomp + cgroups + 网络隔离 |

### 3.4 可扩展性

| 指标 | 要求 |
|---|---|
| 插件热加载 | 无需重启服务 |
| 水平扩展 | 无状态服务支持水平扩缩 |
| 数据库分片 | 按租户分片预留 |

---

## 4. 页面功能 (Web 前端)

### 4.1 页面列表

| 页面 | 路由 | 说明 |
|---|---|---|
| 工作台 | /dashboard | 任务统计、今日概览、快捷操作 |
| 任务列表 | /tasks | 任务筛选、搜索、排序 |
| 任务详情 | /tasks/:id | 任务信息、子任务、评论、活动日志 |
| 目标管理 | /goals | OKR 层级展示 |
| 日历视图 | /calendar | 按日期查看任务 |
| 成员管理 | /members | 成员列表、角色管理 |
| 工作空间设置 | /settings | 插件管理、AI 配置、通知配置 |
| 聊天界面 | /chat | 与 AI 对话，支持结构化卡片 |
| 定时任务 | /schedules | 定时任务管理、执行记录 |
| 业务对象 | /objects/:type | 业务对象实例列表和管理 |

### 4.2 交互组件

| 组件 | 说明 |
|---|---|
| 对话气泡 | 用户消息和 AI 回复 |
| 选择卡片 | ask_human choice 类型 |
| 表单卡片 | ask_human form 类型 |
| 资源卡片 | present_result 渲染的可点击任务/目标/提醒 |
| 文件预览 | 图片、PDF 预览 |
| Markdown 渲染 | AI 回复中的富文本 |

---

## 5. 版本规划

### v1.0 (MVP)
- 工作空间管理
- 会话管理 + 沙箱
- AI Agent 基础对话
- 任务管理（CRUD + 层级 + 标签）
- 文件管理
- 基础 Web 聊天界面

### v1.5
- 业务对象引擎
- 状态机
- 定时任务调度
- ask_human / present_result 交互

### v2.0
- IM 集成（飞书/钉钉）
- 知识库 + RAG
- 插件系统
- 高级权限管理

### v3.0
- EasyOps 运维插件
- 邮件/日历集成
- 跨工作空间协作
- 企业级 SSO
