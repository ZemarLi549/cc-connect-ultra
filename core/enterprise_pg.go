package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

type postgresEnterpriseStore struct {
	db          *sql.DB
	redis       *redis.Client
	redisPrefix string
	rbacTTL     time.Duration
}

type enterpriseDocMeta struct {
	EntityID    string
	TenantID    string
	UserID      string
	OwnerUserID string
	SpaceID     string
	ProjectID   string
	Name        string
	Slug        string
	Scope       string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewEnterpriseDataStore selects the enterprise storage backend.
func NewEnterpriseDataStore(opts EnterpriseStoreOptions) (EnterpriseDataStore, error) {
	if strings.TrimSpace(opts.Postgres.DSN) == "" {
		store := NewEnterpriseStore(opts.FilePath)
		if _, err := store.SaveSettings(opts.SeedSettings); err != nil {
			return nil, err
		}
		return store, nil
	}

	store, err := newPostgresEnterpriseStore(opts)
	if err != nil {
		return nil, err
	}
	if _, err := store.SaveSettings(opts.SeedSettings); err != nil {
		store.Close()
		return nil, err
	}
	return store, nil
}

func newPostgresEnterpriseStore(opts EnterpriseStoreOptions) (*postgresEnterpriseStore, error) {
	db, err := sql.Open("pgx", opts.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if opts.Postgres.MaxOpenConns > 0 {
		db.SetMaxOpenConns(opts.Postgres.MaxOpenConns)
	}
	if opts.Postgres.MaxIdleConns > 0 {
		db.SetMaxIdleConns(opts.Postgres.MaxIdleConns)
	}
	if opts.Postgres.ConnMaxLifetimeSecs > 0 {
		db.SetConnMaxLifetime(time.Duration(opts.Postgres.ConnMaxLifetimeSecs) * time.Second)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	store := &postgresEnterpriseStore{
		db:          db,
		redisPrefix: "cc:enterprise",
		rbacTTL:     5 * time.Minute,
	}
	if prefix := strings.TrimSpace(opts.Redis.KeyPrefix); prefix != "" {
		store.redisPrefix = prefix
	}
	if addr := strings.TrimSpace(opts.Redis.Addr); addr != "" {
		store.redis = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: opts.Redis.Password,
			DB:       opts.Redis.DB,
		})
		if err := store.redis.Ping(context.Background()).Err(); err != nil {
			store.Close()
			return nil, fmt.Errorf("ping redis: %w", err)
		}
	}
	if err := store.initSchema(context.Background()); err != nil {
		store.Close()
		return nil, err
	}
	return store, nil
}

func (s *postgresEnterpriseStore) initSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS enterprise_documents (
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			tenant_id TEXT NOT NULL DEFAULT '',
			user_id TEXT NOT NULL DEFAULT '',
			owner_user_id TEXT NOT NULL DEFAULT '',
			space_id TEXT NOT NULL DEFAULT '',
			project_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT '',
			slug TEXT NOT NULL DEFAULT '',
			scope TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			payload JSONB NOT NULL,
			PRIMARY KEY (entity_type, entity_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_documents_entity_type ON enterprise_documents(entity_type)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_documents_tenant ON enterprise_documents(entity_type, tenant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_documents_user ON enterprise_documents(entity_type, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_documents_owner ON enterprise_documents(entity_type, owner_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_documents_space ON enterprise_documents(entity_type, space_id)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_documents_project ON enterprise_documents(entity_type, project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_documents_scope_status ON enterprise_documents(entity_type, scope, status)`,
		`CREATE TABLE IF NOT EXISTS enterprise_usage_records (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL DEFAULT '',
			user_id TEXT NOT NULL DEFAULT '',
			space_id TEXT NOT NULL DEFAULT '',
			project_name TEXT NOT NULL DEFAULT '',
			provider_name TEXT NOT NULL DEFAULT '',
			model_name TEXT NOT NULL DEFAULT '',
			request_kind TEXT NOT NULL DEFAULT '',
			prompt_tokens BIGINT NOT NULL DEFAULT 0,
			completion_tokens BIGINT NOT NULL DEFAULT 0,
			total_tokens BIGINT NOT NULL DEFAULT 0,
			cost_micros BIGINT NOT NULL DEFAULT 0,
			latency_ms BIGINT NOT NULL DEFAULT 0,
			occurred_at TIMESTAMPTZ NOT NULL,
			payload JSONB NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_usage_tenant ON enterprise_usage_records(tenant_id, occurred_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_usage_user ON enterprise_usage_records(user_id, occurred_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_usage_space ON enterprise_usage_records(space_id, occurred_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_enterprise_usage_provider ON enterprise_usage_records(provider_name, occurred_at DESC)`,
		`CREATE TABLE IF NOT EXISTS enterprise_settings (
			name TEXT PRIMARY KEY,
			payload JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, stmt := range statements {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("init enterprise schema: %w", err)
		}
	}
	return nil
}

func (s *postgresEnterpriseStore) Close() error {
	if s.redis != nil {
		_ = s.redis.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *postgresEnterpriseStore) queryDocs(ctx context.Context, query string, args ...any) ([][]byte, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payloads [][]byte
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, rows.Err()
}

func decodeDocList[T any](payloads [][]byte) ([]T, error) {
	out := make([]T, 0, len(payloads))
	for _, payload := range payloads {
		var item T
		if err := json.Unmarshal(payload, &item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *postgresEnterpriseStore) upsertDocument(ctx context.Context, entityType string, meta enterpriseDocMeta, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}
	if meta.UpdatedAt.IsZero() {
		meta.UpdatedAt = time.Now().UTC()
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO enterprise_documents (
			entity_type, entity_id, tenant_id, user_id, owner_user_id, space_id, project_id,
			name, slug, scope, status, created_at, updated_at, payload
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (entity_type, entity_id) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			user_id = EXCLUDED.user_id,
			owner_user_id = EXCLUDED.owner_user_id,
			space_id = EXCLUDED.space_id,
			project_id = EXCLUDED.project_id,
			name = EXCLUDED.name,
			slug = EXCLUDED.slug,
			scope = EXCLUDED.scope,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at,
			payload = EXCLUDED.payload
	`,
		entityType, meta.EntityID, meta.TenantID, meta.UserID, meta.OwnerUserID, meta.SpaceID, meta.ProjectID,
		meta.Name, meta.Slug, meta.Scope, meta.Status, meta.CreatedAt, meta.UpdatedAt, body,
	)
	return err
}

func (s *postgresEnterpriseStore) listEntities(ctx context.Context, entityType string, clauses []string, args []any, order string) ([][]byte, error) {
	query := `SELECT payload FROM enterprise_documents WHERE entity_type = $1`
	params := []any{entityType}
	params = append(params, args...)
	for _, clause := range clauses {
		query += " AND " + clause
	}
	if order != "" {
		query += " ORDER BY " + order
	}
	return s.queryDocs(ctx, query, params...)
}

func (s *postgresEnterpriseStore) getEntityByID(ctx context.Context, entityType, id string, target any) error {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM enterprise_documents WHERE entity_type = $1 AND entity_id = $2`, entityType, id).Scan(&payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}

func (s *postgresEnterpriseStore) nameForEntity(ctx context.Context, entityType, id string) string {
	if strings.TrimSpace(id) == "" {
		return ""
	}
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM enterprise_documents WHERE entity_type = $1 AND entity_id = $2`, entityType, id).Scan(&name)
	if err != nil {
		return id
	}
	if strings.TrimSpace(name) == "" {
		return id
	}
	return name
}

func (s *postgresEnterpriseStore) bumpRBACVersion(ctx context.Context) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Incr(ctx, s.redisPrefix+":rbac:version").Err()
}

func (s *postgresEnterpriseStore) currentRBACVersion(ctx context.Context) string {
	if s.redis == nil {
		return "0"
	}
	key := s.redisPrefix + ":rbac:version"
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		if err := s.redis.Set(ctx, key, "0", 0).Err(); err == nil {
			return "0"
		}
		return "0"
	}
	if err != nil {
		return "0"
	}
	return val
}

func (s *postgresEnterpriseStore) ListTenants() []EnterpriseTenant {
	ctx := context.Background()
	payloads, err := s.listEntities(ctx, "tenant", nil, nil, "lower(name), created_at")
	if err != nil {
		slog.Error("enterprise postgres list tenants failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseTenant](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode tenants failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertTenant(item EnterpriseTenant) (EnterpriseTenant, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("tenant", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseTenant{}, fmt.Errorf("tenant name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	err := s.upsertDocument(context.Background(), "tenant", enterpriseDocMeta{
		EntityID:  item.ID,
		Name:      item.Name,
		Slug:      item.Slug,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}, item)
	return item, err
}

func (s *postgresEnterpriseStore) ListUsers(tenantID string) []EnterpriseUser {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if tenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, tenantID)
	}
	payloads, err := s.listEntities(ctx, "user", clauses, args, "lower(name), created_at")
	if err != nil {
		slog.Error("enterprise postgres list users failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseUser](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode users failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertUser(item EnterpriseUser) (EnterpriseUser, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("user", item.ID, item.Email+"-"+item.DisplayName)
	item.DisplayName = strings.TrimSpace(item.DisplayName)
	if item.DisplayName == "" {
		item.DisplayName = strings.TrimSpace(item.Email)
	}
	if item.DisplayName == "" {
		return EnterpriseUser{}, fmt.Errorf("user display_name or email is required")
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	err := s.upsertDocument(context.Background(), "user", enterpriseDocMeta{
		EntityID:  item.ID,
		TenantID:  item.TenantID,
		UserID:    item.ID,
		Name:      item.DisplayName,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}, item)
	return item, err
}

func (s *postgresEnterpriseStore) ListSpaces(tenantID, ownerUserID string) []EnterpriseSpace {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if tenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, tenantID)
	}
	if ownerUserID != "" {
		clauses = append(clauses, fmt.Sprintf("owner_user_id = $%d", len(args)+2))
		args = append(args, ownerUserID)
	}
	payloads, err := s.listEntities(ctx, "space", clauses, args, "lower(name), created_at")
	if err != nil {
		slog.Error("enterprise postgres list spaces failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseSpace](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode spaces failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertSpace(item EnterpriseSpace) (EnterpriseSpace, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("space", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseSpace{}, fmt.Errorf("space name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	err := s.upsertDocument(context.Background(), "space", enterpriseDocMeta{
		EntityID:    item.ID,
		TenantID:    item.TenantID,
		OwnerUserID: item.OwnerUserID,
		SpaceID:     item.ID,
		Name:        item.Name,
		Slug:        item.Slug,
		Scope:       item.Visibility,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}, item)
	return item, err
}

func (s *postgresEnterpriseStore) ListSkills(scope, tenantID, ownerUserID string) []EnterpriseSkill {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if scope != "" {
		clauses = append(clauses, fmt.Sprintf("scope = $%d", len(args)+2))
		args = append(args, scope)
	}
	if tenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, tenantID)
	}
	if ownerUserID != "" {
		clauses = append(clauses, fmt.Sprintf("owner_user_id = $%d", len(args)+2))
		args = append(args, ownerUserID)
	}
	payloads, err := s.listEntities(ctx, "skill", clauses, args, "lower(name), created_at")
	if err != nil {
		slog.Error("enterprise postgres list skills failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseSkill](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode skills failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertSkill(item EnterpriseSkill) (EnterpriseSkill, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("skill", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseSkill{}, fmt.Errorf("skill name is required")
	}
	if item.Scope == "" {
		item.Scope = "private"
	}
	if item.Status == "" {
		item.Status = "draft"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	displayName := item.DisplayName
	if displayName == "" {
		displayName = item.Name
	}
	err := s.upsertDocument(context.Background(), "skill", enterpriseDocMeta{
		EntityID:    item.ID,
		TenantID:    item.TenantID,
		OwnerUserID: item.OwnerUserID,
		Name:        displayName,
		Scope:       item.Scope,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}, item)
	return item, err
}

func (s *postgresEnterpriseStore) ListBots(tenantID, ownerUserID string) []EnterpriseBot {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if tenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, tenantID)
	}
	if ownerUserID != "" {
		clauses = append(clauses, fmt.Sprintf("owner_user_id = $%d", len(args)+2))
		args = append(args, ownerUserID)
	}
	payloads, err := s.listEntities(ctx, "bot", clauses, args, "lower(name), created_at")
	if err != nil {
		slog.Error("enterprise postgres list bots failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseBot](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode bots failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertBot(item EnterpriseBot) (EnterpriseBot, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("bot", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseBot{}, fmt.Errorf("bot name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	if item.Scope == "" {
		item.Scope = "tenant"
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	err := s.upsertDocument(context.Background(), "bot", enterpriseDocMeta{
		EntityID:    item.ID,
		TenantID:    item.TenantID,
		OwnerUserID: item.OwnerUserID,
		SpaceID:     item.SpaceID,
		Name:        item.Name,
		Slug:        item.Slug,
		Scope:       item.Scope,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}, item)
	return item, err
}

func (s *postgresEnterpriseStore) ListProviders() []EnterpriseProvider {
	ctx := context.Background()
	payloads, err := s.listEntities(ctx, "provider", nil, nil, "lower(name), created_at")
	if err != nil {
		slog.Error("enterprise postgres list providers failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseProvider](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode providers failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertProvider(item EnterpriseProvider) (EnterpriseProvider, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("provider", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseProvider{}, fmt.Errorf("provider name is required")
	}
	if item.Status == "" {
		item.Status = "enabled"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	name := item.DisplayName
	if name == "" {
		name = item.Name
	}
	err := s.upsertDocument(context.Background(), "provider", enterpriseDocMeta{
		EntityID:  item.ID,
		Name:      name,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}, item)
	return item, err
}

func (s *postgresEnterpriseStore) GetSettings() EnterpriseAIOpsSettings {
	ctx := context.Background()
	var payload []byte
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM enterprise_settings WHERE name = 'aiops'`).Scan(&payload)
	if err == sql.ErrNoRows {
		return EnterpriseAIOpsSettings{}
	}
	if err != nil {
		slog.Error("enterprise postgres get settings failed", "error", err)
		return EnterpriseAIOpsSettings{}
	}
	var item EnterpriseAIOpsSettings
	if err := json.Unmarshal(payload, &item); err != nil {
		slog.Error("enterprise postgres decode settings failed", "error", err)
	}
	return item
}

func (s *postgresEnterpriseStore) SaveSettings(item EnterpriseAIOpsSettings) (EnterpriseAIOpsSettings, error) {
	ctx := context.Background()
	item.UpdatedAt = time.Now().UTC()
	if item.Postgres.Driver == "" {
		item.Postgres.Driver = "postgres"
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return EnterpriseAIOpsSettings{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO enterprise_settings (name, payload, updated_at)
		VALUES ('aiops', $1, $2)
		ON CONFLICT (name) DO UPDATE SET payload = EXCLUDED.payload, updated_at = EXCLUDED.updated_at
	`, payload, item.UpdatedAt)
	return item, err
}

func (s *postgresEnterpriseStore) ListImports(tenantID string) []EnterpriseSkillImportRequest {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if tenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, tenantID)
	}
	payloads, err := s.listEntities(ctx, "import", clauses, args, "created_at DESC")
	if err != nil {
		slog.Error("enterprise postgres list imports failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseSkillImportRequest](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode imports failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) AddImport(item EnterpriseSkillImportRequest) (EnterpriseSkillImportRequest, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("import", item.ID, item.SourceName+"-"+item.SourceRef)
	if item.SourceType == "" {
		item.SourceType = "manual"
	}
	if item.Status == "" {
		item.Status = "queued"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	name := item.SourceName
	if name == "" {
		name = item.SourceRef
	}
	err := s.upsertDocument(context.Background(), "import", enterpriseDocMeta{
		EntityID:    item.ID,
		TenantID:    item.TenantID,
		OwnerUserID: item.OwnerUserID,
		Name:        name,
		Scope:       item.SourceType,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}, item)
	if err != nil {
		return EnterpriseSkillImportRequest{}, err
	}
	settings := s.GetSettings()
	if strings.EqualFold(item.SourceType, "cocoloop") {
		settings.Cocoloop.LastImportAt = now
		_, _ = s.SaveSettings(settings)
	}
	return item, nil
}

func (s *postgresEnterpriseStore) ListUsage(filter EnterpriseUsageFilter) []EnterpriseUsageRecord {
	ctx := context.Background()
	query := `SELECT payload FROM enterprise_usage_records WHERE 1=1`
	args := []any{}
	if filter.TenantID != "" {
		query += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
		args = append(args, filter.TenantID)
	}
	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", len(args)+1)
		args = append(args, filter.UserID)
	}
	if filter.SpaceID != "" {
		query += fmt.Sprintf(" AND space_id = $%d", len(args)+1)
		args = append(args, filter.SpaceID)
	}
	if filter.Provider != "" {
		query += fmt.Sprintf(" AND provider_name = $%d", len(args)+1)
		args = append(args, filter.Provider)
	}
	query += " ORDER BY occurred_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, filter.Limit)
	}
	payloads, err := s.queryDocs(ctx, query, args...)
	if err != nil {
		slog.Error("enterprise postgres list usage failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseUsageRecord](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode usage failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) AddUsage(item EnterpriseUsageRecord) (EnterpriseUsageRecord, error) {
	ctx := context.Background()
	item.ID = ensureEnterpriseID("usage", item.ID)
	if item.OccurredAt.IsZero() {
		item.OccurredAt = time.Now().UTC()
	}
	if item.TotalTokens == 0 {
		item.TotalTokens = item.PromptTokens + item.CompletionTokens
	}
	payload, err := json.Marshal(item)
	if err != nil {
		return EnterpriseUsageRecord{}, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO enterprise_usage_records (
			id, tenant_id, user_id, space_id, project_name, provider_name, model_name, request_kind,
			prompt_tokens, completion_tokens, total_tokens, cost_micros, latency_ms, occurred_at, payload
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15
		)
	`, item.ID, item.TenantID, item.UserID, item.SpaceID, item.ProjectName, item.ProviderName, item.ModelName, item.RequestKind,
		item.PromptTokens, item.CompletionTokens, item.TotalTokens, item.CostMicros, item.LatencyMs, item.OccurredAt, payload)
	if err != nil {
		return EnterpriseUsageRecord{}, err
	}
	return item, nil
}

func (s *postgresEnterpriseStore) Overview() EnterpriseOverview {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `SELECT entity_type, COUNT(*) FROM enterprise_documents GROUP BY entity_type`)
	if err != nil {
		slog.Error("enterprise postgres overview count query failed", "error", err)
		return EnterpriseOverview{}
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var entityType string
		var count int
		if err := rows.Scan(&entityType, &count); err != nil {
			slog.Error("enterprise postgres overview scan failed", "error", err)
			return EnterpriseOverview{}
		}
		counts[entityType] = count
	}

	var usageCount int
	var totalTokens, totalCost int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cost_micros), 0) FROM enterprise_usage_records`).Scan(&usageCount, &totalTokens, &totalCost); err != nil {
		slog.Error("enterprise postgres overview usage query failed", "error", err)
	}

	return EnterpriseOverview{
		TenantsCount:    counts["tenant"],
		UsersCount:      counts["user"],
		SpacesCount:     counts["space"],
		SkillsCount:     counts["skill"],
		BotsCount:       counts["bot"],
		RolesCount:      counts["role"],
		ProjectsCount:   counts["project"],
		TasksCount:      counts["task"],
		ProvidersCount:  counts["provider"],
		ImportsCount:    counts["import"],
		UsageCount:      usageCount,
		TotalTokens:     totalTokens,
		TotalCostMicros: totalCost,
	}
}

func (s *postgresEnterpriseStore) Leaderboard(groupBy string, limit int) []EnterpriseLeaderboardEntry {
	ctx := context.Background()
	if groupBy == "" {
		groupBy = "user"
	}
	if limit <= 0 {
		limit = 20
	}

	groupColumn := "user_id"
	entityType := "user"
	switch groupBy {
	case "tenant":
		groupColumn = "tenant_id"
		entityType = "tenant"
	case "space":
		groupColumn = "space_id"
		entityType = "space"
	default:
		groupBy = "user"
	}

	query := fmt.Sprintf(`
		SELECT %s, COUNT(*), COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cost_micros), 0)
		FROM enterprise_usage_records
		WHERE %s <> ''
		GROUP BY %s
		ORDER BY COALESCE(SUM(total_tokens), 0) DESC, %s ASC
		LIMIT $1
	`, groupColumn, groupColumn, groupColumn, groupColumn)

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		slog.Error("enterprise postgres leaderboard query failed", "error", err)
		return nil
	}
	defer rows.Close()

	var out []EnterpriseLeaderboardEntry
	for rows.Next() {
		var subjectID string
		var requests, tokens, costMicros int64
		if err := rows.Scan(&subjectID, &requests, &tokens, &costMicros); err != nil {
			slog.Error("enterprise postgres leaderboard scan failed", "error", err)
			return nil
		}
		out = append(out, EnterpriseLeaderboardEntry{
			SubjectType: groupBy,
			SubjectID:   subjectID,
			SubjectName: s.nameForEntity(ctx, entityType, subjectID),
			Requests:    requests,
			Tokens:      tokens,
			CostMicros:  costMicros,
		})
	}
	return out
}

func (s *postgresEnterpriseStore) ListPermissions() []EnterprisePermission {
	return BuiltinEnterprisePermissions()
}

func (s *postgresEnterpriseStore) ListRoles(tenantID, scope string) []EnterpriseRole {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if tenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, tenantID)
	}
	if scope != "" {
		clauses = append(clauses, fmt.Sprintf("scope = $%d", len(args)+2))
		args = append(args, scope)
	}
	payloads, err := s.listEntities(ctx, "role", clauses, args, "lower(name), created_at")
	if err != nil {
		slog.Error("enterprise postgres list roles failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseRole](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode roles failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertRole(item EnterpriseRole) (EnterpriseRole, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("role", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseRole{}, fmt.Errorf("role name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	if item.Scope == "" {
		item.Scope = "tenant"
	}
	if item.Status == "" {
		item.Status = "active"
	}
	item.PermissionIDs = normalizePermissionIDs(item.PermissionIDs)
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	err := s.upsertDocument(context.Background(), "role", enterpriseDocMeta{
		EntityID:  item.ID,
		TenantID:  item.TenantID,
		Name:      item.Name,
		Slug:      item.Slug,
		Scope:     item.Scope,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}, item)
	if err == nil {
		s.bumpRBACVersion(context.Background())
	}
	return item, err
}

func (s *postgresEnterpriseStore) ListRoleBindings(filter EnterpriseRoleBindingFilter) []EnterpriseRoleBinding {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if filter.TenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, filter.TenantID)
	}
	if filter.UserID != "" {
		clauses = append(clauses, fmt.Sprintf("user_id = $%d", len(args)+2))
		args = append(args, filter.UserID)
	}
	if filter.SpaceID != "" {
		clauses = append(clauses, fmt.Sprintf("space_id = $%d", len(args)+2))
		args = append(args, filter.SpaceID)
	}
	if filter.ProjectID != "" {
		clauses = append(clauses, fmt.Sprintf("project_id = $%d", len(args)+2))
		args = append(args, filter.ProjectID)
	}
	if filter.Scope != "" {
		clauses = append(clauses, fmt.Sprintf("scope = $%d", len(args)+2))
		args = append(args, filter.Scope)
	}
	payloads, err := s.listEntities(ctx, "role_binding", clauses, args, "created_at")
	if err != nil {
		slog.Error("enterprise postgres list role bindings failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseRoleBinding](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode role bindings failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertRoleBinding(item EnterpriseRoleBinding) (EnterpriseRoleBinding, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("binding", item.ID, item.RoleID+"-"+item.UserID+"-"+item.SpaceID+"-"+item.ProjectID)
	if strings.TrimSpace(item.RoleID) == "" {
		return EnterpriseRoleBinding{}, fmt.Errorf("role_id is required")
	}
	if strings.TrimSpace(item.UserID) == "" {
		return EnterpriseRoleBinding{}, fmt.Errorf("user_id is required")
	}
	if item.Scope == "" {
		item.Scope = "tenant"
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	err := s.upsertDocument(context.Background(), "role_binding", enterpriseDocMeta{
		EntityID:  item.ID,
		TenantID:  item.TenantID,
		UserID:    item.UserID,
		SpaceID:   item.SpaceID,
		ProjectID: item.ProjectID,
		Name:      item.RoleID,
		Scope:     item.Scope,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}, item)
	if err == nil {
		s.bumpRBACVersion(context.Background())
	}
	return item, err
}

func (s *postgresEnterpriseStore) ResolveAccess(req EnterpriseAccessRequest) (EnterpriseAccessProfile, error) {
	if strings.TrimSpace(req.UserID) == "" {
		return EnterpriseAccessProfile{}, fmt.Errorf("user_id is required")
	}
	ctx := context.Background()
	version := s.currentRBACVersion(ctx)
	cacheKey := fmt.Sprintf("%s:rbac:effective:%s:%s:%s:%s:%s", s.redisPrefix, version, req.TenantID, req.UserID, req.SpaceID, req.ProjectID)
	if s.redis != nil {
		if cached, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
			var profile EnterpriseAccessProfile
			if jsonErr := json.Unmarshal([]byte(cached), &profile); jsonErr == nil {
				return profile, nil
			}
		}
	}

	roleClauses := []string{}
	roleArgs := []any{}
	if req.TenantID != "" {
		roleClauses = append(roleClauses, fmt.Sprintf("(tenant_id = $%d OR tenant_id = '')", len(roleArgs)+2))
		roleArgs = append(roleArgs, req.TenantID)
	}
	rolePayloads, err := s.listEntities(ctx, "role", roleClauses, roleArgs, "lower(name), created_at")
	if err != nil {
		return EnterpriseAccessProfile{}, err
	}
	roles, err := decodeDocList[EnterpriseRole](rolePayloads)
	if err != nil {
		return EnterpriseAccessProfile{}, err
	}

	bindingArgs := []any{req.UserID}
	bindingClauses := []string{"user_id = $2"}
	if req.TenantID != "" {
		bindingClauses = append(bindingClauses, fmt.Sprintf("(tenant_id = $%d OR tenant_id = '')", len(bindingArgs)+2))
		bindingArgs = append(bindingArgs, req.TenantID)
	}
	bindingPayloads, err := s.listEntities(ctx, "role_binding", bindingClauses, bindingArgs, "created_at")
	if err != nil {
		return EnterpriseAccessProfile{}, err
	}
	bindings, err := decodeDocList[EnterpriseRoleBinding](bindingPayloads)
	if err != nil {
		return EnterpriseAccessProfile{}, err
	}
	profile := resolveEnterpriseAccessFromData(req, roles, bindings)
	if s.redis != nil {
		if body, err := json.Marshal(profile); err == nil {
			_ = s.redis.Set(ctx, cacheKey, body, s.rbacTTL).Err()
		}
	}
	return profile, nil
}

func (s *postgresEnterpriseStore) ListProjects(tenantID, spaceID string) []EnterpriseProjectProfile {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if tenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, tenantID)
	}
	if spaceID != "" {
		clauses = append(clauses, fmt.Sprintf("space_id = $%d", len(args)+2))
		args = append(args, spaceID)
	}
	payloads, err := s.listEntities(ctx, "project", clauses, args, "lower(name), created_at")
	if err != nil {
		slog.Error("enterprise postgres list projects failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseProjectProfile](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode projects failed", "error", err)
	}
	return items
}

func (s *postgresEnterpriseStore) UpsertProject(item EnterpriseProjectProfile) (EnterpriseProjectProfile, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("project", item.ID, item.Name)
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return EnterpriseProjectProfile{}, fmt.Errorf("project name is required")
	}
	if item.Slug == "" {
		item.Slug = enterpriseSlug(item.Name, item.ID)
	}
	if item.Source == "" {
		item.Source = "ui"
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	err := s.upsertDocument(context.Background(), "project", enterpriseDocMeta{
		EntityID:    item.ID,
		TenantID:    item.TenantID,
		OwnerUserID: item.OwnerUserID,
		SpaceID:     item.SpaceID,
		ProjectID:   item.ID,
		Name:        item.Name,
		Slug:        item.Slug,
		Scope:       item.Source,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}, item)
	return item, err
}

func (s *postgresEnterpriseStore) SyncProjects(items []EnterpriseProjectProfile) error {
	for _, item := range items {
		item.Source = "config"
		if _, err := s.UpsertProject(item); err != nil {
			return err
		}
	}
	return nil
}

func (s *postgresEnterpriseStore) ListTasks(filter EnterpriseTaskFilter) []EnterpriseTask {
	ctx := context.Background()
	clauses, args := []string{}, []any{}
	if filter.TenantID != "" {
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)+2))
		args = append(args, filter.TenantID)
	}
	if filter.SpaceID != "" {
		clauses = append(clauses, fmt.Sprintf("space_id = $%d", len(args)+2))
		args = append(args, filter.SpaceID)
	}
	if filter.OwnerUserID != "" {
		clauses = append(clauses, fmt.Sprintf("owner_user_id = $%d", len(args)+2))
		args = append(args, filter.OwnerUserID)
	}
	if filter.AssigneeUserID != "" {
		clauses = append(clauses, fmt.Sprintf("user_id = $%d", len(args)+2))
		args = append(args, filter.AssigneeUserID)
	}
	if filter.TaskType != "" {
		clauses = append(clauses, fmt.Sprintf("scope = $%d", len(args)+2))
		args = append(args, filter.TaskType)
	}
	if filter.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)+2))
		args = append(args, filter.Status)
	}
	payloads, err := s.listEntities(ctx, "task", clauses, args, "updated_at DESC")
	if err != nil {
		slog.Error("enterprise postgres list tasks failed", "error", err)
		return nil
	}
	items, err := decodeDocList[EnterpriseTask](payloads)
	if err != nil {
		slog.Error("enterprise postgres decode tasks failed", "error", err)
		return nil
	}
	filtered := make([]EnterpriseTask, 0, len(items))
	for _, item := range items {
		if filter.Priority != "" && item.Priority != filter.Priority {
			continue
		}
		if filter.Tag != "" {
			found := false
			for _, tag := range item.Tags {
				if strings.EqualFold(strings.TrimSpace(tag), strings.TrimSpace(filter.Tag)) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if q := strings.TrimSpace(strings.ToLower(filter.Query)); q != "" {
			if !strings.Contains(strings.ToLower(item.Title), q) && !strings.Contains(strings.ToLower(item.Description), q) {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func (s *postgresEnterpriseStore) UpsertTask(item EnterpriseTask) (EnterpriseTask, error) {
	now := time.Now().UTC()
	item.ID = ensureStableEnterpriseID("task", item.ID, item.Title)
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" {
		return EnterpriseTask{}, fmt.Errorf("task title is required")
	}
	if item.TaskType == "" {
		item.TaskType = "task"
	}
	if item.Priority == "" {
		item.Priority = "medium"
	}
	if item.Status == "" {
		item.Status = "todo"
	}
	item.Tags = normalizeTagList(item.Tags)
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if item.Status == "done" && item.CompletedAt.IsZero() {
		item.CompletedAt = now
	}
	err := s.upsertDocument(context.Background(), "task", enterpriseDocMeta{
		EntityID:    item.ID,
		TenantID:    item.TenantID,
		UserID:      item.AssigneeUserID,
		OwnerUserID: item.OwnerUserID,
		SpaceID:     item.SpaceID,
		ProjectID:   item.ParentTaskID,
		Name:        item.Title,
		Scope:       item.TaskType,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}, item)
	return item, err
}
