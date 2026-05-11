package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	mysqldrv "github.com/go-sql-driver/mysql"
)

type mysqlRefreshStore struct {
	db *sql.DB
}

// probeMySQL 用短超时探测 MySQL 是否可连接（忽略库不存在，等同 ping 通）。
func probeMySQL(dsn string) error {
	cfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("解析 DSN: %w", err)
	}
	cfg.Timeout = 2 * time.Second
	cfg.ReadTimeout = 2 * time.Second
	cfg.WriteTimeout = 2 * time.Second
	cfg.DBName = "" // 库可能尚未创建，先只验证服务端可达
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func newMySQLRefreshStore(dsn string) (*mysqlRefreshStore, error) {
	if err := ensureMySQLDatabase(dsn); err != nil {
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
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
  token_hash   CHAR(64)     NOT NULL PRIMARY KEY,
  payload      TEXT         NOT NULL,
  expires_at   BIGINT       NOT NULL,
  created_at   BIGINT       NOT NULL,
  KEY idx_auth_refresh_exp (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`); err != nil {
		_ = db.Close()
		return nil, err
	}
	_, _ = db.ExecContext(ctx, `DELETE FROM auth_refresh_tokens WHERE expires_at < ?`, time.Now().Unix())
	return &mysqlRefreshStore{db: db}, nil
}

type userRecordPersist struct {
	Username string   `json:"username"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
}

func (s *mysqlRefreshStore) tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *mysqlRefreshStore) newToken(user UserRecord) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	hash := s.tokenHash(token)
	payload, err := json.Marshal(userRecordPersist{
		Username: user.Username,
		TenantID: user.TenantID,
		Roles:    user.Roles,
	})
	if err != nil {
		return "", err
	}
	now := time.Now()
	exp := now.Add(refreshTTL())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO auth_refresh_tokens(token_hash, payload, expires_at, created_at) VALUES(?,?,?,?)`,
		hash, string(payload), exp.Unix(), now.Unix(),
	)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *mysqlRefreshStore) consume(token string) (UserRecord, bool) {
	hash := s.tokenHash(token)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return UserRecord{}, false
	}
	defer func() { _ = tx.Rollback() }()

	var payload string
	var exp int64
	err = tx.QueryRowContext(ctx,
		`SELECT payload, expires_at FROM auth_refresh_tokens WHERE token_hash = ? FOR UPDATE`, hash,
	).Scan(&payload, &exp)
	if err != nil {
		return UserRecord{}, false
	}
	if time.Now().Unix() > exp {
		_, _ = tx.ExecContext(ctx, `DELETE FROM auth_refresh_tokens WHERE token_hash = ?`, hash)
		_ = tx.Commit()
		return UserRecord{}, false
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM auth_refresh_tokens WHERE token_hash = ?`, hash); err != nil {
		return UserRecord{}, false
	}
	if err := tx.Commit(); err != nil {
		return UserRecord{}, false
	}
	var up userRecordPersist
	if err := json.Unmarshal([]byte(payload), &up); err != nil {
		return UserRecord{}, false
	}
	return UserRecord{
		Username: up.Username,
		Password: "",
		TenantID: up.TenantID,
		Roles:    up.Roles,
	}, true
}

func (s *mysqlRefreshStore) revoke(token string) {
	hash := s.tokenHash(token)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = s.db.ExecContext(ctx, `DELETE FROM auth_refresh_tokens WHERE token_hash = ?`, hash)
}

// ensureMySQLDatabase 解析 DSN 中的 DBName，若不存在则尝试创建（utf8mb4）。
func ensureMySQLDatabase(dsn string) error {
	cfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("解析 MySQL DSN 失败: %w", err)
	}
	if cfg.DBName == "" {
		return nil
	}
	dbName := cfg.DBName
	cfg.DBName = ""
	adminDSN := cfg.FormatDSN()
	admin, err := sql.Open("mysql", adminDSN)
	if err != nil {
		return err
	}
	defer func() { _ = admin.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		return fmt.Errorf("连接 MySQL 失败: %w", err)
	}
	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		dbName,
	)
	if _, err := admin.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("创建数据库 %s 失败: %w", dbName, err)
	}
	return nil
}
