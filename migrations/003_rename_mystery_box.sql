PRAGMA foreign_keys = OFF;

BEGIN;

INSERT INTO game_types (code,name,frontend_module,is_active)
VALUES ('lucky-dip','Lucky Dip','games/lucky-dip.js',1)
ON CONFLICT(code) DO UPDATE SET
    name=excluded.name,
    frontend_module=excluded.frontend_module,
    is_active=1;

UPDATE campaigns
SET game_type_code='lucky-dip',
    updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE game_type_code='mystery-box';

UPDATE campaign_games
SET game_type_code='lucky-dip',
    updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE game_type_code='mystery-box'
  AND NOT EXISTS (
      SELECT 1 FROM campaign_games existing
      WHERE existing.campaign_id=campaign_games.campaign_id
        AND existing.game_type_code='lucky-dip'
  );

DELETE FROM campaign_games WHERE game_type_code='mystery-box';
DELETE FROM game_types WHERE code='mystery-box';

COMMIT;

PRAGMA foreign_keys = ON;
