# Hermes + cc-connect Enterprise Topology

## Goal

Use a single shared Hermes platform as the enterprise-facing AI control plane,
while keeping each user's runtime context, workspace, and session isolated.

This document answers the practical question:

> Can an enterprise share one Hermes agent platform underneath everything?

The answer is:

- yes, as a shared control plane and routing layer
- no, if it means every user shares one runtime context, one session, or one work directory

The correct architecture is:

**one Hermes platform, many isolated space runtimes**

## Recommended Layering

### Layer 1: Enterprise Entry

This is the user-facing entry layer:

- web portal
- chat platform bots
- internal assistant portal
- browser extension
- IM integrations

This layer authenticates the user and forwards requests to Hermes.

### Layer 2: Hermes Control Plane

Hermes should own the shared enterprise logic:

- tenant and user identity
- RBAC and policy
- model catalog
- skill catalog
- routing rules
- token metering
- ranking
- audit
- quota checks
- provider selection policy

Hermes is the enterprise "brain" and governance layer.

Hermes should **not** be the single shared agent session for all users.

### Layer 3: cc-connect Runtime Orchestrator

`cc-connect` is a strong runtime layer for:

- platform message handling
- workspace routing
- per-space session lifecycle
- per-space agent process management
- tool execution and streaming
- skill discovery and prompt injection

In this combined architecture:

- Hermes decides **who** the user is and **what** policy applies
- `cc-connect` decides **which workspace runtime** should execute the request

### Layer 4: Space Runtime

Each user or team space should have its own runtime boundary:

- dedicated workspace directory
- dedicated session history
- dedicated provider/model defaults
- dedicated skill overlay
- dedicated usage attribution

Recommended runtime unit:

- one agent subprocess per active space

Future stronger isolation options:

- one OS user per tenant/user
- one container per space
- one VM or sandbox group per high-sensitivity department

### Layer 5: Model Provider Layer

This is where the real model calls happen:

- DeepSeek
- GLM
- Qwen
- Claude-compatible gateways
- OpenAI-compatible internal proxies

This layer should remain replaceable.

Hermes should manage the provider catalog and switching policy, while
`cc-connect` executes the selected runtime configuration for the space.

## The Wrong Architecture

Do **not** do this:

```text
All users
  -> One Hermes bot
  -> One agent process
  -> One session history
  -> One shared workspace
```

This fails in practice because:

- context leaks between users
- workspace files become shared accidentally
- long conversations pollute unrelated requests
- skills and memory become cross-user
- token usage cannot be attributed precisely
- permission approvals become ambiguous
- enterprise audit becomes weak

## The Right Architecture

Do this instead:

```text
Users / Teams
  -> Hermes control plane
  -> cc-connect runtime router
  -> Space A runtime -> Provider X
  -> Space B runtime -> Provider Y
  -> Space C runtime -> Provider X
```

Each space stays logically isolated even though the platform is centrally managed.

## Deployment Topology

### Minimal Production Topology

```text
[Web / IM / Portal]
        |
        v
[Hermes API + Admin UI]
        |
        v
[cc-connect Runtime Cluster]
        |
        +--> [Workspace Store]
        +--> [Skill Store]
        +--> [Session Store]
        +--> [Usage Store]
        |
        +--> [DeepSeek / GLM / Other Providers]
```

### Recommended Service Responsibilities

#### Hermes service

- SSO / identity
- tenant, user, space metadata
- provider catalog and policy
- skill publishing workflow
- usage aggregation
- leaderboard generation
- admin APIs and UI

#### cc-connect service

- channel or request ingress
- workspace resolution
- per-space agent spawn/resume/cleanup
- tool and session orchestration
- skill injection into runtime
- streaming reply handling

#### Shared data stores

- metadata DB
- object/file storage for skills and assets
- session state store
- usage/event store

## How Hermes Should Integrate With cc-connect

The cleanest integration model is:

1. Hermes is the source of truth for tenants, users, spaces, and providers
2. Hermes calls or embeds `cc-connect` management/runtime APIs
3. `cc-connect` resolves a request into a concrete space runtime
4. `cc-connect` runs the agent and reports usage back to Hermes

Two integration options are reasonable.

### Option A: Hermes above cc-connect

Hermes is a separate upstream service.

Flow:

1. user request enters Hermes
2. Hermes authenticates user
3. Hermes resolves tenant/user/space/provider policy
4. Hermes forwards execution request to `cc-connect`
5. `cc-connect` runs the space runtime
6. `cc-connect` returns usage and execution metadata

Use this when:

- you want a real enterprise platform
- you need SSO, RBAC, billing, audits
- you expect a web console and external integrations

### Option B: Hermes logic embedded into cc-connect

Hermes is a product concept, but most logic lives inside this codebase.

Flow:

1. request enters `cc-connect`
2. enterprise APIs and metadata are served in-process
3. workspace and provider selection happen in-process
4. usage is persisted locally or to a shared DB

Use this when:

- you want a fast MVP
- you control deployment tightly
- you want to ship the first enterprise version quickly

The current branch is moving toward Option B first, with a path to Option A later.

## Space Model

Each enterprise user should work through a Space object.

Recommended fields:

- `tenant_id`
- `owner_user_id`
- `space_id`
- `workspace_dir`
- `project_name`
- `current_provider`
- `current_model`
- `visibility`
- `shared_skill_ids`

This is the key abstraction that lets one Hermes platform serve many isolated users.

## Skill Architecture

Hermes should manage skill metadata and visibility:

- private skills
- tenant shared skills
- public or curated skills

`cc-connect` should materialize the effective skill set for a space:

1. system skills
2. public skills
3. tenant skills
4. user private skills

That merged view is what the runtime sees.

## Provider Switching Model

Hermes should own:

- provider definitions
- enabled models
- defaults
- policy rules

`cc-connect` should apply the final selected provider/model into the runtime.

Recommended selection priority:

1. request override
2. active space selection
3. user preference
4. tenant default
5. system default

## Usage, Token Metering, and Ranking

Hermes should aggregate usage at the following dimensions:

- tenant
- user
- space
- provider
- model
- request kind

Suggested outputs:

- daily token dashboards
- per-team rankings
- monthly usage reports
- provider cost comparisons
- hot spaces / dormant spaces

`cc-connect` should emit raw usage records close to execution time.
Hermes should aggregate and present them.

## Isolation Recommendations

### MVP isolation

- one workspace directory per space
- one runtime session per space
- one usage stream per space

### Stronger enterprise isolation

- one subprocess per active space
- no shared workdir
- no shared session file
- provider keys kept server-side only

### High-security deployment

- one OS user or container per tenant or per space
- filesystem ACLs
- stronger audit and approval controls

## Recommended Rollout Path

### Stage 1: Shared Hermes metadata + isolated spaces

Build first:

- tenant/user/space/provider/usage APIs
- multi-workspace routing
- per-space runtime isolation
- simple skill visibility scopes

### Stage 2: Provider policy and usage accounting

Add:

- per-space provider switching
- request-level usage capture
- token leaderboard
- quotas

### Stage 3: Skill publishing and enterprise governance

Add:

- skill drafts and publishing
- tenant/public visibility
- audit log
- approval flow

### Stage 4: Hardening

Add:

- SSO
- RBAC
- DB-backed persistence
- stronger sandboxing
- runtime clusters

## Concrete Next Code Steps

The next valuable implementation steps in this repo are:

1. bind `EnterpriseSpace` to multi-workspace runtime resolution
2. let a space carry active provider/model defaults
3. emit real usage records from agent completions
4. build effective skill overlays by `system / tenant / user`
5. expose simple web UI views on top of the enterprise APIs

## Final Recommendation

Use Hermes as the enterprise shared platform.

Do not use one shared Hermes runtime session for everybody.

The safe and scalable pattern is:

**shared Hermes control plane + isolated per-space runtimes powered by cc-connect**
