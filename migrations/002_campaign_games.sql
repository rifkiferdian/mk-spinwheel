PRAGMA foreign_keys = ON;

-- Satu campaign dapat menyediakan beberapa jenis game. Hadiah dan stok tetap
-- dimiliki campaign agar semua game memakai alokasi hadiah yang sama.
CREATE TABLE IF NOT EXISTS campaign_games (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id    INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    game_type_code TEXT NOT NULL REFERENCES game_types(code) ON DELETE RESTRICT,
    game_config    TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(game_config)),
    is_active      INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    display_order  INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (campaign_id, game_type_code)
);

CREATE INDEX IF NOT EXISTS idx_campaign_games_campaign
    ON campaign_games (campaign_id, is_active, display_order);

-- Campaign lama otomatis memperoleh game yang sebelumnya dipilih.
INSERT OR IGNORE INTO campaign_games (
    campaign_id, game_type_code, game_config, is_active, display_order
)
SELECT id, game_type_code, game_config, 1, 0
FROM campaigns;
