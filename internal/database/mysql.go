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
	return db, nil
}
