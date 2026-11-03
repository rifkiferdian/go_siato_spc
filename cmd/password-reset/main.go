package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"siato-spc/internal/auth"
	"siato-spc/internal/config"
	"siato-spc/internal/database"
)

func main() {
	user := flag.String("user", "", "email atau username user")
	flag.Parse()
	if strings.TrimSpace(*user) == "" {
		log.Fatal("gunakan: go run ./cmd/password-reset -user user@example.com")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	service := auth.NewService(db, cfg.Environment == "production", cfg.PasswordPepper)
	token, err := service.CreatePasswordResetToken(context.Background(), *user)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Tautan reset berlaku 30 menit:\n%s/reset-password?token=%s\n", strings.TrimRight(cfg.BaseURL, "/"), token)
}
