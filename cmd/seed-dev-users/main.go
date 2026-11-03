package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"siato-spc/internal/auth"
	"siato-spc/internal/config"
	"siato-spc/internal/database"
)

type seedUser struct {
	name     string
	emailKey string
	passKey  string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if !strings.EqualFold(cfg.Environment, "development") {
		log.Fatal("seeder password hanya boleh dijalankan dengan APP_ENV=development")
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	users := []seedUser{
		{"Super Admin", "SeedUsers__SuperAdmin__Email", "SeedUsers__SuperAdmin__Password"},
		{"Admin Keuangan", "SeedUsers__AdminKeuangan__Email", "SeedUsers__AdminKeuangan__Password"},
		{"Manager Keuangan", "SeedUsers__ManagerKeuangan__Email", "SeedUsers__ManagerKeuangan__Password"},
		{"Rektor", "SeedUsers__Rektor__Email", "SeedUsers__Rektor__Password"},
		{"Staff Akademik", "SeedUsers__StaffAkademik__Email", "SeedUsers__StaffAkademik__Password"},
		{"Marketing / PMB", "SeedUsers__MarketingPmb__Email", "SeedUsers__MarketingPmb__Password"},
	}
	for _, user := range users {
		if os.Getenv(user.emailKey) == "" || os.Getenv(user.passKey) == "" {
			log.Fatalf("%s dan %s wajib diisi", user.emailKey, user.passKey)
		}
	}

	service := auth.NewService(db, false, cfg.PasswordPepper)
	for _, user := range users {
		if err := service.SetDevelopmentPassword(context.Background(), os.Getenv(user.emailKey), os.Getenv(user.passKey)); err != nil {
			log.Fatalf("gagal memperbarui %s: %v", user.name, err)
		}
		fmt.Printf("[OK] %s\n", user.name)
	}
	fmt.Println("Semua password development berhasil dibuat dengan Argon2id.")
}
