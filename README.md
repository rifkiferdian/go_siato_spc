# SIATO SPC — Go

Fondasi rewrite SIATO SPC menggunakan Go, Gin, MySQL, Tailwind CSS 4, Roboto, dan Font Awesome.

## Menjalankan lokal

```powershell
Copy-Item .env.example .env
npm.cmd install
go mod download
go install github.com/cosmtrek/air@v1.49.0
air
```

Aplikasi tersedia di `http://localhost:8080`. Air memantau perubahan file Go, template HTML, dan sumber CSS. Sebelum setiap rebuild, Air juga membangun ulang Tailwind CSS. Pastikan database MySQL pada `.env` sudah dibuat dan dapat diakses. Endpoint `GET /health` memeriksa koneksi aplikasi dan database.

## Reset password user

Semua user hasil migrasi wajib membuat password Go baru. Buat tautan reset sekali pakai dengan:

```powershell
go run ./cmd/password-reset -user superadmin@siato.local
```

Tautan berlaku 30 menit. Password baru menggunakan Argon2id dengan salt acak dan pepper dari `AUTH_PASSWORD_PEPPER`. Jangan mengganti pepper setelah user membuat password karena seluruh hash akan menjadi tidak dapat diverifikasi.

Untuk mengisi ulang password keenam akun development dari variabel `SeedUsers__...`:

```powershell
go run ./cmd/seed-dev-users
```

Command tersebut hanya dapat dijalankan ketika `APP_ENV=development`.

## Build

```powershell
npm.cmd run build
```

Binary Windows akan dibuat pada `bin/siato-spc.exe`. Jalankan binary dari root folder proyek karena template dan static assets saat ini dibaca dari folder `web`.

## Struktur

```text
cmd/server/             entry point aplikasi
internal/config/        konfigurasi environment
internal/database/      koneksi dan pool MySQL
internal/httpserver/    router dan HTTP handler
web/templates/          Go HTML templates
web/styles/             sumber Tailwind CSS
web/static/             CSS dan aset hasil build
scripts/                script frontend/build
```
