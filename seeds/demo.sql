PRAGMA foreign_keys = ON;

-- Kompatibilitas dengan seed demo sebelumnya.
UPDATE campaigns
SET name='Festival Hadiah Ceria', slug='festival-hadiah-ceria',
    game_config='{"theme":"carnival","duration_ms":5200,"show_confetti":true,"headline":"Putar & Menangkan Hadiah!"}',
    is_active=1, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
WHERE slug='demo-game-hadiah'
  AND NOT EXISTS (SELECT 1 FROM campaigns WHERE slug='festival-hadiah-ceria');

-- Idempotent: aman dijalankan ulang tanpa menggandakan hadiah.
INSERT INTO campaigns (game_type_code,name,slug,game_config,is_active)
VALUES ('wheel','Festival Hadiah Ceria','festival-hadiah-ceria',
        '{"theme":"carnival","duration_ms":5200,"show_confetti":true,"headline":"Putar & Menangkan Hadiah!"}',1)
ON CONFLICT(slug) DO UPDATE SET
    game_type_code=excluded.game_type_code, name=excluded.name,
    game_config=excluded.game_config, is_active=excluded.is_active,
    updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now');

INSERT OR IGNORE INTO campaign_games (campaign_id,game_type_code,game_config,is_active,display_order)
SELECT id,'wheel',game_config,1,1 FROM campaigns WHERE slug='festival-hadiah-ceria';

INSERT INTO campaign_games (campaign_id,game_type_code,game_config,is_active,display_order)
SELECT id,'claw',
       '{"theme":"manna-claw","duration_ms":5200,"show_confetti":true,"headline":"Capit & Bawa Pulang Hadiahnya!"}',
       1,2
FROM campaigns WHERE slug='festival-hadiah-ceria'
ON CONFLICT(campaign_id,game_type_code) DO UPDATE SET
    game_config=excluded.game_config,is_active=1,display_order=excluded.display_order;

-- 1. Voucher Rp10.000
UPDATE prizes SET description='Voucher belanja senilai Rp10.000',color='#F97316',weight=30,
 initial_stock=100,remaining_stock=MAX(0,100-(SELECT COUNT(*) FROM game_results r WHERE r.prize_id=prizes.id)),is_unlimited=0,
 requires_claim=1,display_order=1,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='festival-hadiah-ceria') AND name='Voucher Rp10.000';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Voucher Rp10.000','Voucher belanja senilai Rp10.000','#F97316',30,100,100,0,1,1 FROM campaigns
WHERE slug='festival-hadiah-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Voucher Rp10.000');

-- 2. Kopi Gratis
UPDATE prizes SET description='Satu minuman kopi pilihan',color='#292524',weight=25,
 initial_stock=50,remaining_stock=MAX(0,50-(SELECT COUNT(*) FROM game_results r WHERE r.prize_id=prizes.id)),is_unlimited=0,
 requires_claim=1,display_order=2,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='festival-hadiah-ceria') AND name='Kopi Gratis';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Kopi Gratis','Satu minuman kopi pilihan','#292524',25,50,50,0,1,2 FROM campaigns
WHERE slug='festival-hadiah-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Kopi Gratis');

-- 3. Merchandise Eksklusif
UPDATE prizes SET name='Merchandise Eksklusif',description='Merchandise edisi khusus event',
 color='#C2410C',weight=10,initial_stock=20,remaining_stock=MAX(0,20-(SELECT COUNT(*) FROM game_results r WHERE r.prize_id=prizes.id)),
 is_unlimited=0,requires_claim=1,display_order=3,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='festival-hadiah-ceria')
  AND name IN ('Merchandise','Merchandise Eksklusif');
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Merchandise Eksklusif','Merchandise edisi khusus event','#C2410C',10,20,20,0,1,3 FROM campaigns
WHERE slug='festival-hadiah-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Merchandise Eksklusif');

-- 4. Voucher Rp50.000
UPDATE prizes SET description='Voucher belanja spesial senilai Rp50.000',color='#7C2D12',weight=5,
 initial_stock=10,remaining_stock=MAX(0,10-(SELECT COUNT(*) FROM game_results r WHERE r.prize_id=prizes.id)),is_unlimited=0,
 requires_claim=1,display_order=4,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='festival-hadiah-ceria') AND name='Voucher Rp50.000';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Voucher Rp50.000','Voucher belanja spesial senilai Rp50.000','#7C2D12',5,10,10,0,1,4 FROM campaigns
WHERE slug='festival-hadiah-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Voucher Rp50.000');

-- 5. Belum Beruntung (stok tanpa batas, tidak perlu klaim)
UPDATE prizes SET description='Tetap semangat dan coba lagi',color='#64748B',weight=30,
 initial_stock=0,remaining_stock=0,is_unlimited=1,requires_claim=0,
 display_order=5,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='festival-hadiah-ceria') AND name='Belum Beruntung';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Belum Beruntung','Tetap semangat dan coba lagi','#64748B',30,0,0,1,0,5 FROM campaigns
WHERE slug='festival-hadiah-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Belum Beruntung');
