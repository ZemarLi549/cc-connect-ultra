# Enterprise AI Workspace Platform Design

## Goal

Build an enterprise internal AI office platform on top of `cc-connect` with:

- A single shared platform deployment
- Per-user isolated spaces and sessions
- Tenant-scoped and public skill sharing
- Multi-model switching across providers like DeepSeek and GLM
- Token metering, cost tracking, and leaderboards
- A management API and future web control plane

This document defines the first practical architecture for evolving `cc-connect`
into a product similar to an internal "Molili"-style AI workspace system.

For the concrete placement of a shared Hermes platform in this architecture,
see [Hermes + cc-connect Enterprise Topology](hermes-cc-connect-topology.md).

## Product Model

The platform uses five core business objects:

1. Tenant
   A company or organization using the platform.

2. User
   A human member inside a tenant.

3. Space
   An isolated workspace owned by a user or team. A space maps to a concrete
   workspace directory and its own AI session/runtime state.

4. Skill
   A reusable instruction package with optional versioning and sharing scope.

5. Provider
   A model/provider configuration exposed for user or space-level switching.

## Key Principle

The platform is shared, but execution is isolated.

That means:

- One deployment can serve many tenants and users
- Each user/space gets its own workspace directory
- Each user/space gets its own session history
- Each user/space should eventually get its own agent subprocess
- Shared skills and provider catalogs are control-plane resources, not shared runtime state

This avoids the biggest enterprise risks: cross-user context leakage, file
access leaks, and mixed session history.

## Reuse From Existing cc-connect

The current codebase already provides strong foundations:

- `multi-workspace` mode routes channels to workspaces
- Per-workspace agent/session pools already exist in `Engine`
- Skills are auto-discovered from `SKILL.md`
- Management API exists and can host a web control plane
- Providers already support project-level and global configuration

The enterprise platform should extend these existing primitives, not replace them.

## Runtime Architecture

### Control Plane

The control plane owns enterprise metadata:

- tenants
- users
- spaces
- skill registry
- provider catalog
- usage records
- quotas
- rankings

This layer is exposed through the management API and later through a web UI.

### Data Plane

The data plane is still `cc-connect`:

- receives messages from chat platforms or web clients
- resolves a user/space
- loads the correct workspace
- selects a provider/model
- runs the agent
- streams results back

### Workspace Isolation

Recommended directory layout:

```text
<base_dir>/
  <tenant_slug>/
    <user_id>/
      <space_slug>/
        repo/
        .cc-connect/
        .claude/
        .workspace-meta/
```

Each space should eventually map to:

- isolated `work_dir`
- isolated session store
- isolated skill overlay
- isolated usage/accounting context

## Skill Sharing Model

Skills should support three scopes:

- `private`: only the owner can use it
- `tenant`: visible inside the tenant
- `public`: visible to all tenants or globally curated catalog

Recommended effective merge order at runtime:

1. system skills
2. public skills
3. tenant skills
4. user private skills

Conflict rule:

- later layers override earlier layers by skill name

This makes it possible for enterprises to customize shared skills while still
keeping a curated public catalog.

## Multi-Model Switching

Providers should become first-class enterprise resources.

Each provider entry may include:

- provider name
- provider type
- base URL
- supported model list
- default model
- status
- metadata/tags

Examples:

- DeepSeek
- GLM
- Qwen
- OpenAI-compatible internal gateways

The switching rules should eventually support:

- tenant default provider/model
- user default provider/model
- space default provider/model
- per-request override

Priority recommendation:

1. request override
2. space selection
3. user default
4. tenant default
5. system default

## Token Metering and Ranking

Usage accounting should happen per request/turn with:

- tenant id
- user id
- space id
- provider
- model
- prompt tokens
- completion tokens
- total tokens
- cost in micros
- latency
- timestamp

This enables:

- cost dashboards
- quota alerts
- tenant cost allocation
- user ranking
- space ranking
- provider/model usage analysis

The first implementation can be file-backed JSON. A future production upgrade
should move this to a database.

## Security and Isolation Requirements

Minimum production requirements:

- separate workspace dirs per user/space
- no shared session history between users
- skill publishing must be auditable
- provider keys must not leak to ordinary users
- all privileged config mutations must be authenticated

Recommended future hardening:

- per-space subprocess execution
- OS-user or container isolation for high-trust deployments
- approval workflow for shared/public skill publishing
- signed skill bundles for curated marketplaces

## Phase Plan

### Phase 1: Control-Plane Skeleton

Deliver:

- enterprise store
- management API endpoints
- tenant/user/space/skill/provider/usage models
- overview and leaderboard APIs

This phase is implemented in the current branch.

### Phase 2: Space Runtime Integration

Deliver:

- mapping enterprise spaces to workspace bindings
- per-space provider selection
- usage record emission from real model runs
- skill scope overlay resolution

### Phase 3: Web Console

Deliver:

- admin dashboard
- user space list
- skill publishing UI
- provider management UI
- usage dashboards and leaderboard views

### Phase 4: Enterprise Hardening

Deliver:

- quota enforcement
- RBAC
- approval workflow
- audit logs
- stronger runtime isolation
- public skill marketplace governance

## Management API Additions

The first enterprise management endpoints are:

- `GET /api/v1/enterprise/overview`
- `GET|POST /api/v1/enterprise/tenants`
- `GET|POST /api/v1/enterprise/users`
- `GET|POST /api/v1/enterprise/spaces`
- `GET|POST /api/v1/enterprise/skills`
- `GET|POST /api/v1/enterprise/providers`
- `GET|POST /api/v1/enterprise/usage`
- `GET /api/v1/enterprise/leaderboard`

These endpoints are intentionally simple and file-backed for the first slice.

## Immediate Next Steps

1. Wire enterprise spaces to workspace binding and multi-workspace routing
2. Add provider selection per space and user
3. Emit usage records from actual agent completions
4. Implement skill scope overlay on top of `SkillRegistry`
5. Build a lightweight web UI against the new enterprise endpoints

## Non-Goals For Phase 1

The first slice does not yet implement:

- real auth/SSO
- DB-backed persistence
- billing enforcement
- container sandboxing
- full skill publishing workflow
- live provider switching in the UI

Those should follow after the runtime and control-plane contracts are stable.
