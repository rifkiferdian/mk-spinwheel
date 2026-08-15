# PrizePlay

Platform mini-game berhadiah dengan Go, SQLite, Go HTML templates, JavaScript,
dan Tailwind CSS.

## Menjalankan aplikasi

```powershell
npm install
npm run css:build
go run ./cmd/server
```

Buka <http://localhost:8080/admin>. Jika database belum mempunyai admin,
aplikasi akan membuka halaman pembuatan admin pertama. Setelah akun pertama
dibuat, halaman setup otomatis tidak dapat digunakan lagi.

Halaman permainan demo tersedia di:

```text
http://localhost:8080/play/festival-hadiah-ceria
```

Campaign demo mempunyai lima hasil: Voucher Rp10.000, Kopi Gratis,
Merchandise Eksklusif, Voucher Rp50.000, dan Belum Beruntung. Untuk menerapkan
atau memperbarui data demo secara aman:

```powershell
go run ./cmd/dbinit -seed
```

Database default berada di `data/game.db`. Server otomatis menjalankan schema
yang belum ada tanpa menghapus data.

## Development dengan Air

Pastikan Air sudah terpasang:

```powershell
go install github.com/air-verse/air@latest
```

Jalankan dua terminal. Terminal pertama memantau perubahan Go, template HTML,
dan migration SQL:

```powershell
air
```

Terminal kedua membangun ulang Tailwind ketika class pada template berubah:

```powershell
npm run css:dev
```

Konfigurasi Air berada di `.air.toml`. Hasil build sementara disimpan di `tmp/`
dan otomatis dibersihkan ketika Air dihentikan.

## Environment

| Nama | Default | Fungsi |
|---|---|---|
| `APP_ADDR` | `:8080` | Alamat HTTP server |
| `DATABASE_PATH` | `data/game.db` | Lokasi database SQLite |
| `SECURE_COOKIE` | `false` | Isi `true` ketika aplikasi memakai HTTPS |

Untuk production, gunakan HTTPS dan set `SECURE_COOKIE=true`.
