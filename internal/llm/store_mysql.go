package llm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
)

const ddlModelProfile = `CREATE TABLE IF NOT EXISTS model_profile (
  id                     VARCHAR(64)  NOT NULL PRIMARY KEY,
  name                   VARCHAR(128) NOT NULL,
  purpose                VARCHAR(16)  NOT NULL,
  provider               VARCHAR(32)  NOT NULL,
  endpoint               VARCHAR(512) NOT NULL,
  model                  VARCHAR(128) NOT NULL,
  api_key_ref            VARCHAR(128) NOT NULL DEFAULT '',
  timeout_seconds        INT          NOT NULL DEFAULT 180,
  max_retries            INT          NOT NULL DEFAULT 2,
  retry_backoff_ms       INT          NOT NULL DEFAULT 400,
  rpm                    INT          NOT NULL DEFAULT 0,
  stream_timeout_seconds INT          NOT NULL DEFAULT 0,
  enabled                TINYINT(1)   NOT NULL DEFAULT 1,
  created_by             VARCHAR(64)  NOT NULL DEFAULT '',
  created_at             BIGINT       NOT NULL,
  updated_at             BIGINT       NOT NULL,
  KEY idx_purpose_enabled (purpose, enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

const ddlTenantModelBinding = `CREATE TABLE IF NOT EXISTS tenant_model_binding (
  tenant_id    VARCHAR(64)  NOT NULL,
  profile_id   VARCHAR(64)  NOT NULL,
  purpose      VARCHAR(16)  NOT NULL,
  is_default   TINYINT(1)   NOT NULL DEFAULT 0,
  rpm_override INT          NOT NULL DEFAULT 0,
  created_at   BIGINT       NOT NULL,
  PRIMARY KEY (tenant_id, profile_id),
  KEY idx_tenant_purpose_default (tenant_id, purpose, is_default),
  CONSTRAINT fk_tmb_profile FOREIGN KEY (profile_id) REFERENCES model_profile(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

type mysqlProfileStore struct {
	mu sync.Mutex
	db *sql.DB
}

func probeRegistryMySQL(dsn string) error {
	cfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("解析 DSN: %w", err)
	}
	cfg.Timeout = 2 * time.Second
	cfg.ReadTimeout = 2 * time.Second
	cfg.WriteTimeout = 2 * time.Second
	cfg.DBName = ""
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func ensureRegistryDatabase(dsn string) error {
	cfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("解析 DSN: %w", err)
	}
	if cfg.DBName == "" {
		return nil
	}
	dbName := cfg.DBName
	cfg.DBName = ""
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer func() { _ = admin.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		return err
	}
	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		dbName,
	)
	_, err = admin.ExecContext(ctx, stmt)
	return err
}

func newMySQLProfileStore(dsn string) (*mysqlProfileStore, error) {
	if err := ensureRegistryDatabase(dsn); err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, ddl := range []string{ddlModelProfile, ddlTenantModelBinding} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return &mysqlProfileStore{db: db}, nil
}

func (s *mysqlProfileStore) Kind() string { return "mysql" }

func (s *mysqlProfileStore) scanProfile(scanner interface {
	Scan(...interface{}) error
}) (Profile, error) {
	var (
		p                          Profile
		purposeStr, providerStr    string
		enabled                    int
		createdAtSec, updatedAtSec int64
	)
	if err := scanner.Scan(
		&p.ID, &p.Name, &purposeStr, &providerStr, &p.Endpoint, &p.Model, &p.APIKeyRef,
		&p.TimeoutSeconds, &p.MaxRetries, &p.RetryBackoffMs, &p.RPM, &p.StreamTimeoutSeconds,
		&enabled, &p.CreatedBy, &createdAtSec, &updatedAtSec,
	); err != nil {
		return Profile{}, err
	}
	p.Purpose = Purpose(purposeStr)
	p.Provider = ProviderKind(providerStr)
	p.Enabled = enabled == 1
	p.CreatedAt = time.Unix(createdAtSec, 0)
	p.UpdatedAt = time.Unix(updatedAtSec, 0)
	return p, nil
}

const selectProfileCols = `id,name,purpose,provider,endpoint,model,api_key_ref,
		timeout_seconds,max_retries,retry_backoff_ms,rpm,stream_timeout_seconds,
		enabled,created_by,created_at,updated_at`

func (s *mysqlProfileStore) ListProfiles() ([]Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+selectProfileCols+" FROM model_profile ORDER BY purpose, id")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Profile
	for rows.Next() {
		p, err := s.scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *mysqlProfileStore) GetProfile(id string) (Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	row := s.db.QueryRowContext(ctx,
		"SELECT "+selectProfileCols+" FROM model_profile WHERE id = ?", id)
	p, err := s.scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	return p, err
}

func (s *mysqlProfileStore) UpsertProfile(p Profile) error {
	now := time.Now().Unix()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Unix(now, 0)
	}
	enabled := 0
	if p.Enabled {
		enabled = 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, `INSERT INTO model_profile
		(id,name,purpose,provider,endpoint,model,api_key_ref,
		 timeout_seconds,max_retries,retry_backoff_ms,rpm,stream_timeout_seconds,
		 enabled,created_by,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
		  name=VALUES(name), purpose=VALUES(purpose), provider=VALUES(provider),
		  endpoint=VALUES(endpoint), model=VALUES(model), api_key_ref=VALUES(api_key_ref),
		  timeout_seconds=VALUES(timeout_seconds), max_retries=VALUES(max_retries),
		  retry_backoff_ms=VALUES(retry_backoff_ms), rpm=VALUES(rpm),
		  stream_timeout_seconds=VALUES(stream_timeout_seconds),
		  enabled=VALUES(enabled), updated_at=VALUES(updated_at)`,
		p.ID, p.Name, string(p.Purpose), string(p.Provider), p.Endpoint, p.Model, p.APIKeyRef,
		p.TimeoutSeconds, p.MaxRetries, p.RetryBackoffMs, p.RPM, p.StreamTimeoutSeconds,
		enabled, p.CreatedBy, p.CreatedAt.Unix(), now,
	)
	return err
}

func (s *mysqlProfileStore) DeleteProfile(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var cnt int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tenant_model_binding WHERE profile_id = ?", id,
	).Scan(&cnt); err != nil {
		return err
	}
	if cnt > 0 {
		return ErrProfileInUse
	}
	res, err := s.db.ExecContext(ctx, "DELETE FROM model_profile WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProfileNotFound
	}
	return nil
}

func (s *mysqlProfileStore) ListBindings(tenantID string) ([]TenantBinding, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx,
		`SELECT tenant_id, profile_id, purpose, is_default, rpm_override, created_at
		 FROM tenant_model_binding WHERE tenant_id = ?
		 ORDER BY purpose, profile_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []TenantBinding
	for rows.Next() {
		var (
			b            TenantBinding
			purposeStr   string
			isDefault    int
			createdAtSec int64
		)
		if err := rows.Scan(&b.TenantID, &b.ProfileID, &purposeStr, &isDefault, &b.RPMOverride, &createdAtSec); err != nil {
			return nil, err
		}
		b.Purpose = Purpose(purposeStr)
		b.IsDefault = isDefault == 1
		b.CreatedAt = time.Unix(createdAtSec, 0)
		out = append(out, b)
	}
	return out, rows.Err()
}

// SetBindings 整体替换某租户的绑定（一次事务，避免半截状态；同一 purpose 仅允许一个 is_default=1）。
func (s *mysqlProfileStore) SetBindings(tenantID string, bs []TenantBinding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 服务端再校验一次：同一 purpose 最多一个 default。
	defaultByPurpose := map[Purpose]int{}
	for _, b := range bs {
		if b.IsDefault {
			defaultByPurpose[b.Purpose]++
		}
	}
	for p, n := range defaultByPurpose {
		if n > 1 {
			return fmt.Errorf("purpose=%s 仅允许一个 is_default=1", p)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM tenant_model_binding WHERE tenant_id = ?", tenantID); err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, b := range bs {
		var pCount int
		if err := tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM model_profile WHERE id = ?", b.ProfileID,
		).Scan(&pCount); err != nil {
			return err
		}
		if pCount == 0 {
			return fmt.Errorf("%w: id=%s", ErrProfileNotFound, b.ProfileID)
		}
		isDefault := 0
		if b.IsDefault {
			isDefault = 1
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO tenant_model_binding(tenant_id, profile_id, purpose, is_default, rpm_override, created_at)
			 VALUES(?,?,?,?,?,?)`,
			tenantID, b.ProfileID, string(b.Purpose), isDefault, b.RPMOverride, now,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *mysqlProfileStore) DefaultBinding(tenantID string, purpose Purpose) (TenantBinding, Profile, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var (
		b            TenantBinding
		purposeStr   string
		isDefault    int
		createdAtSec int64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT tenant_id, profile_id, purpose, is_default, rpm_override, created_at
		 FROM tenant_model_binding
		 WHERE tenant_id = ? AND purpose = ? AND is_default = 1
		 LIMIT 1`,
		tenantID, string(purpose),
	).Scan(&b.TenantID, &b.ProfileID, &purposeStr, &isDefault, &b.RPMOverride, &createdAtSec)
	if errors.Is(err, sql.ErrNoRows) {
		return TenantBinding{}, Profile{}, ErrBindingNotFound
	}
	if err != nil {
		return TenantBinding{}, Profile{}, err
	}
	b.Purpose = Purpose(purposeStr)
	b.IsDefault = isDefault == 1
	b.CreatedAt = time.Unix(createdAtSec, 0)

	prof, err := s.GetProfile(b.ProfileID)
	if err != nil {
		return TenantBinding{}, Profile{}, err
	}
	if !prof.Enabled {
		return TenantBinding{}, Profile{}, fmt.Errorf("默认 profile %s 已禁用", prof.ID)
	}
	return b, prof, nil
}
