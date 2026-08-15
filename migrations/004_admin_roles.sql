PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS admin_roles (
    admin_id   INTEGER PRIMARY KEY REFERENCES admins(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'super_admin'
               CHECK (role IN ('super_admin', 'campaign_admin')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- Seluruh akun yang sudah ada tetap memperoleh akses penuh.
INSERT OR IGNORE INTO admin_roles (admin_id,role)
SELECT id,'super_admin' FROM admins;
