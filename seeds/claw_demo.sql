PRAGMA foreign_keys = ON;

INSERT INTO campaigns (game_type_code,name,slug,game_config,is_active)
VALUES ('claw','Capit Boneka Ceria','capit-boneka-ceria',
        '{"theme":"manna-claw","duration_ms":5200,"show_confetti":true,"headline":"Capit & Bawa Pulang Hadiahnya!"}',1)
ON CONFLICT(slug) DO UPDATE SET
    game_type_code=excluded.game_type_code,name=excluded.name,
    game_config=excluded.game_config,is_active=excluded.is_active,
    updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now');

INSERT OR IGNORE INTO campaign_games (campaign_id,game_type_code,game_config,is_active,display_order)
SELECT id,'claw',game_config,1,1 FROM campaigns WHERE slug='capit-boneka-ceria';

UPDATE prizes SET description='Boneka beruang lembut edisi Manna',color='#F97316',weight=12,
 initial_stock=15,remaining_stock=MAX(0,15-(SELECT COUNT(*) FROM game_results r WHERE r.prize_id=prizes.id)),
 is_unlimited=0,requires_claim=1,display_order=1,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='capit-boneka-ceria') AND name='Boneka Beruang';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Boneka Beruang','Boneka beruang lembut edisi Manna','#F97316',12,15,15,0,1,1 FROM campaigns
WHERE slug='capit-boneka-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Boneka Beruang');

UPDATE prizes SET description='Boneka panda lucu dan menggemaskan',color='#292524',weight=8,
 initial_stock=8,remaining_stock=MAX(0,8-(SELECT COUNT(*) FROM game_results r WHERE r.prize_id=prizes.id)),
 is_unlimited=0,requires_claim=1,display_order=2,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='capit-boneka-ceria') AND name='Boneka Panda';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Boneka Panda','Boneka panda lucu dan menggemaskan','#292524',8,8,8,0,1,2 FROM campaigns
WHERE slug='capit-boneka-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Boneka Panda');

UPDATE prizes SET description='Boneka kelinci lembut bertelinga panjang',color='#FB923C',weight=10,
 initial_stock=12,remaining_stock=MAX(0,12-(SELECT COUNT(*) FROM game_results r WHERE r.prize_id=prizes.id)),
 is_unlimited=0,requires_claim=1,display_order=3,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='capit-boneka-ceria') AND name='Boneka Kelinci';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Boneka Kelinci','Boneka kelinci lembut bertelinga panjang','#FB923C',10,12,12,0,1,3 FROM campaigns
WHERE slug='capit-boneka-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Boneka Kelinci');

UPDATE prizes SET description='Voucher belanja Manna senilai Rp10.000',color='#FACC15',weight=30,
 initial_stock=50,remaining_stock=MAX(0,50-(SELECT COUNT(*) FROM game_results r WHERE r.prize_id=prizes.id)),
 is_unlimited=0,requires_claim=1,display_order=4,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='capit-boneka-ceria') AND name='Voucher Rp10.000';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Voucher Rp10.000','Voucher belanja Manna senilai Rp10.000','#FACC15',30,50,50,0,1,4 FROM campaigns
WHERE slug='capit-boneka-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Voucher Rp10.000');

UPDATE prizes SET description='Capitnya belum membawa hadiah, coba lagi',color='#64748B',weight=40,
 initial_stock=0,remaining_stock=0,is_unlimited=1,requires_claim=0,display_order=5,is_active=1
WHERE campaign_id=(SELECT id FROM campaigns WHERE slug='capit-boneka-ceria') AND name='Belum Beruntung';
INSERT INTO prizes (campaign_id,name,description,color,weight,initial_stock,remaining_stock,is_unlimited,requires_claim,display_order)
SELECT id,'Belum Beruntung','Capitnya belum membawa hadiah, coba lagi','#64748B',40,0,0,1,0,5 FROM campaigns
WHERE slug='capit-boneka-ceria' AND NOT EXISTS
(SELECT 1 FROM prizes p WHERE p.campaign_id=campaigns.id AND p.name='Belum Beruntung');
