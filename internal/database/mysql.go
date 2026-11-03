package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"siato-spc/internal/config"
)

func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	dsnConfig := mysql.Config{
		User:                 cfg.User,
		Passwd:               cfg.Password,
		Net:                  "tcp",
		Addr:                 cfg.Host + ":" + cfg.Port,
		DBName:               cfg.Name,
		ParseTime:            true,
		Loc:                  time.UTC,
		AllowNativePasswords: true,
	}
	dsn := dsnConfig.FormatDSN()

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrasi database Go: %w", err)
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	userColumns := []struct {
		name       string
		definition string
	}{
		{"GoPasswordHash", "LONGTEXT NULL"},
		{"GoPasswordResetRequired", "TINYINT(1) NOT NULL DEFAULT 1"},
		{"GoPasswordUpdatedAt", "DATETIME(6) NULL"},
	}
	for _, column := range userColumns {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'users' AND column_name = ?`, column.name).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN `"+column.name+"` "+column.definition); err != nil {
				return err
			}
		}
	}
	legacyUserColumns := []string{"PasswordHash", "ConcurrencyStamp"}
	for _, column := range legacyUserColumns {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'users' AND column_name = ?`, column).Scan(&count); err != nil {
			return err
		}
		if count == 1 {
			if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN `"+column+"`"); err != nil {
				return err
			}
		}
	}

	const sessionsTable = `
CREATE TABLE IF NOT EXISTS go_auth_sessions (
    token_hash CHAR(64) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    csrf_token CHAR(64) NOT NULL,
    security_stamp TEXT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    last_seen_at DATETIME(6) NOT NULL,
    remember_me TINYINT(1) NOT NULL DEFAULT 0,
    ip_address VARCHAR(45) NULL,
    user_agent VARCHAR(500) NULL,
    PRIMARY KEY (token_hash),
    KEY IX_go_auth_sessions_user_id (user_id),
    KEY IX_go_auth_sessions_expires_at (expires_at),
    CONSTRAINT FK_go_auth_sessions_users_user_id
        FOREIGN KEY (user_id) REFERENCES users (Id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci`
	_, err := db.ExecContext(ctx, sessionsTable)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE s FROM go_auth_sessions s JOIN users u ON u.Id = s.user_id WHERE u.GoPasswordResetRequired = 1`); err != nil {
		return err
	}
	const resetTokensTable = `
CREATE TABLE IF NOT EXISTS go_password_reset_tokens (
    token_hash CHAR(64) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    used_at DATETIME(6) NULL,
    PRIMARY KEY (token_hash),
    KEY IX_go_password_reset_tokens_user_id (user_id),
    KEY IX_go_password_reset_tokens_expires_at (expires_at),
    CONSTRAINT FK_go_password_reset_tokens_users_user_id
        FOREIGN KEY (user_id) REFERENCES users (Id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci`
	_, err = db.ExecContext(ctx, resetTokensTable)
	return err
}
