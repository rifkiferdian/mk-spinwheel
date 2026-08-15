# Database SQLite

Database ini tidak menyimpan customer. Setiap permainan hanya mempunyai token
session anonim untuk memastikan satu sesi menghasilkan maksimal satu hasil.

## Relasi utama

```text
game_types  1 ─── n campaigns 1 ─── n prizes
                         │                │
                         ├── n game_sessions
                         │          │
                         │          └── 1 game_results ── 1 prizes
                         │
                         └── n access_codes (opsional)
```

## Aturan stok

- `weight` menentukan peluang relatif, bukan persentase wajib.
- `is_unlimited = 1` digunakan untuk hasil tanpa batas seperti "Belum Beruntung".
- Insert ke `game_results` otomatis mengurangi `remaining_stock`.
- Database menolak hasil jika stok hadiah sudah habis.
- Satu `game_session_id` hanya boleh mempunyai satu hasil.
- Pemilihan hadiah di aplikasi Go tetap harus dilakukan dalam satu transaksi.

## Membuat database dengan Go

Project menyediakan utilitas inisialisasi sehingga `sqlite3` CLI tidak wajib
terpasang:

```powershell
go run ./cmd/dbinit
```

Untuk ikut memasukkan campaign dan hadiah contoh:

```powershell
go run ./cmd/dbinit -seed
```

Lokasi database dapat diganti dengan parameter `-db`.

## Membuat database dengan SQLite CLI

Dengan SQLite CLI:

```powershell
New-Item -ItemType Directory -Force data
sqlite3 ./data/game.db ".read ./migrations/001_initial_schema.sql"
```

Data contoh bersifat opsional:

```powershell
sqlite3 ./data/game.db ".read ./seeds/demo.sql"
```

Jangan memasukkan database produksi ke Git. Simpan migration SQL sebagai sumber
struktur database dan buat file database ketika aplikasi pertama kali dijalankan.
