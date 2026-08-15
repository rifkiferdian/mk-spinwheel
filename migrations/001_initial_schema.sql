PRAGMA foreign_keys = ON;

-- Jenis permainan disimpan sebagai data agar game baru dapat ditambahkan
-- tanpa mengubah struktur tabel campaign.
CREATE TABLE IF NOT EXISTS game_types (
    code            TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    frontend_module TEXT NOT NULL,
    is_active       INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS admins (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL COLLATE NOCASE UNIQUE,
    password_hash TEXT NOT NULL,
    is_active     INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS campaigns (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    game_type_code TEXT NOT NULL REFERENCES game_types(code),
    name           TEXT NOT NULL,
    slug           TEXT NOT NULL COLLATE NOCASE UNIQUE,
    game_config    TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(game_config)),
    starts_at      TEXT,
    ends_at        TEXT,
    is_active      INTEGER NOT NULL DEFAULT 0 CHECK (is_active IN (0, 1)),
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (ends_at IS NULL OR starts_at IS NULL OR ends_at > starts_at)
);

-- Opsional: digunakan jika satu permainan harus memakai kupon/QR/kode struk.
-- Tidak ada data atau ID customer yang disimpan.
CREATE TABLE IF NOT EXISTS access_codes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    code        TEXT NOT NULL COLLATE NOCASE,
    status      TEXT NOT NULL DEFAULT 'unused'
                CHECK (status IN ('unused', 'used', 'expired', 'disabled')),
    used_at     TEXT,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (campaign_id, code)
);

CREATE TABLE IF NOT EXISTS prizes (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id     INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    description     TEXT,
    image_path      TEXT,
    color           TEXT,
    weight          REAL NOT NULL DEFAULT 1 CHECK (weight > 0),
    initial_stock   INTEGER NOT NULL DEFAULT 0 CHECK (initial_stock >= 0),
    remaining_stock INTEGER NOT NULL DEFAULT 0 CHECK (remaining_stock >= 0),
    is_unlimited    INTEGER NOT NULL DEFAULT 0 CHECK (is_unlimited IN (0, 1)),
    requires_claim  INTEGER NOT NULL DEFAULT 1 CHECK (requires_claim IN (0, 1)),
    display_order   INTEGER NOT NULL DEFAULT 0,
    is_active       INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (is_unlimited = 1 OR remaining_stock <= initial_stock)
);

-- Sesi bersifat anonim. Token acak mencegah satu klik/reload menghasilkan
-- lebih dari satu hadiah tanpa perlu mengenali customer.
CREATE TABLE IF NOT EXISTS game_sessions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    campaign_id    INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE RESTRICT,
    access_code_id INTEGER UNIQUE REFERENCES access_codes(id) ON DELETE RESTRICT,
    session_token  TEXT NOT NULL UNIQUE,
    status         TEXT NOT NULL DEFAULT 'created'
                   CHECK (status IN ('created', 'playing', 'completed', 'expired', 'cancelled')),
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    started_at     TEXT,
    completed_at   TEXT,
    expires_at     TEXT
);

CREATE TABLE IF NOT EXISTS game_results (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    game_session_id INTEGER NOT NULL UNIQUE REFERENCES game_sessions(id) ON DELETE RESTRICT,
    campaign_id     INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE RESTRICT,
    prize_id        INTEGER NOT NULL REFERENCES prizes(id) ON DELETE RESTRICT,
    claim_code      TEXT COLLATE NOCASE UNIQUE,
    claim_status    TEXT NOT NULL DEFAULT 'pending'
                    CHECK (claim_status IN ('pending', 'claimed', 'not_required', 'cancelled')),
    played_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    claimed_at      TEXT,
    CHECK (
        (claim_status = 'claimed' AND claimed_at IS NOT NULL)
        OR (claim_status <> 'claimed' AND claimed_at IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_campaigns_active
    ON campaigns (is_active, starts_at, ends_at);

CREATE INDEX IF NOT EXISTS idx_prizes_available
    ON prizes (campaign_id, is_active, is_unlimited, remaining_stock);

CREATE INDEX IF NOT EXISTS idx_sessions_campaign_created
    ON game_sessions (campaign_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_results_campaign_played
    ON game_results (campaign_id, played_at DESC);

CREATE INDEX IF NOT EXISTS idx_results_claim_status
    ON game_results (claim_status, played_at DESC);

-- Menolak hasil jika session/prize bukan milik campaign yang sama atau stok habis.
CREATE TRIGGER IF NOT EXISTS validate_game_result_before_insert
BEFORE INSERT ON game_results
BEGIN
    SELECT CASE
        WHEN (SELECT campaign_id FROM game_sessions WHERE id = NEW.game_session_id) <> NEW.campaign_id
        THEN RAISE(ABORT, 'session does not belong to campaign')
    END;

    SELECT CASE
        WHEN (SELECT campaign_id FROM prizes WHERE id = NEW.prize_id) <> NEW.campaign_id
        THEN RAISE(ABORT, 'prize does not belong to campaign')
    END;

    SELECT CASE
        WHEN EXISTS (
            SELECT 1 FROM prizes
            WHERE id = NEW.prize_id
              AND is_unlimited = 0
              AND remaining_stock <= 0
        )
        THEN RAISE(ABORT, 'prize stock is empty')
    END;
END;

-- Pengurangan stok dan penyelesaian session terjadi dalam transaksi insert hasil.
CREATE TRIGGER IF NOT EXISTS complete_game_after_result_insert
AFTER INSERT ON game_results
BEGIN
    UPDATE prizes
       SET remaining_stock = remaining_stock - 1,
           updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
     WHERE id = NEW.prize_id
       AND is_unlimited = 0;

    UPDATE game_sessions
       SET status = 'completed',
           completed_at = NEW.played_at
     WHERE id = NEW.game_session_id;

    UPDATE access_codes
       SET status = 'used',
           used_at = NEW.played_at
     WHERE id = (SELECT access_code_id
                   FROM game_sessions
                  WHERE id = NEW.game_session_id);
END;

-- Identitas hasil undian tidak boleh diganti setelah tercatat.
CREATE TRIGGER IF NOT EXISTS keep_game_result_immutable
BEFORE UPDATE OF game_session_id, campaign_id, prize_id, played_at ON game_results
BEGIN
    SELECT RAISE(ABORT, 'game result identity is immutable');
END;

CREATE VIEW IF NOT EXISTS campaign_prize_summary AS
SELECT
    c.id AS campaign_id,
    c.name AS campaign_name,
    p.id AS prize_id,
    p.name AS prize_name,
    p.initial_stock,
    p.remaining_stock,
    p.is_unlimited,
    COUNT(r.id) AS total_won,
    SUM(CASE WHEN r.claim_status = 'claimed' THEN 1 ELSE 0 END) AS total_claimed
FROM campaigns c
JOIN prizes p ON p.campaign_id = c.id
LEFT JOIN game_results r ON r.prize_id = p.id
GROUP BY c.id, p.id;

INSERT OR IGNORE INTO game_types (code, name, frontend_module) VALUES
    ('wheel',       'Spin Wheel',      'games/wheel.js'),
    ('fishing',     'Pancing Hadiah',  'games/fishing.js'),
    ('claw',        'Capit Hadiah',    'games/claw.js'),
    ('lucky-dip',   'Lucky Dip',       'games/lucky-dip.js');
