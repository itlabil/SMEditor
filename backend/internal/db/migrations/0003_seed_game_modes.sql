INSERT OR IGNORE INTO game_modes (code, name, genre, block_code, terms_json, sort_order) VALUES
('mlbb', 'Mobile Legends', 'moba', 'moba', '{"objektif":"Turtle, Lord","bangunan":"Turret, base","penghargaan":"MVP"}', 1),
('hok', 'Honor of Kings', 'moba', 'moba', '{"objektif":"Tyrant, Overlord, Tempest Dragon","bangunan":"Tower, crystal","penghargaan":"MVP"}', 2),
('aov', 'Arena of Valor', 'moba', 'moba', '{"objektif":"Abyssal Dragon, Dark Slayer","bangunan":"Tower, core","penghargaan":"MVP"}', 3),
('wildrift', 'Wild Rift', 'moba', 'moba', '{"objektif":"Dragon, Rift Herald, Baron Nashor","bangunan":"Turret, Nexus","penghargaan":"MVP"}', 4),
('lol', 'League of Legends', 'moba', 'moba', '{"objektif":"Dragon, Rift Herald, Baron Nashor, Elder Dragon","bangunan":"Turret, inhibitor, Nexus","penghargaan":"Player of the Game"}', 5),
('dota2', 'Dota 2', 'moba', 'moba', '{"objektif":"Roshan dan Aegis, Tormentor","bangunan":"Tower, barracks, Ancient","penghargaan":"Statistik akhir dan net worth"}', 6),

('pubgm', 'PUBG Mobile', 'br', 'br', '{"istilah_menang":"Winner Winner Chicken Dinner"}', 7),
('freefire', 'Free Fire', 'br', 'br', '{"istilah_menang":"Booyah"}', 8),

('valorant', 'Valorant', 'fps', 'fps', '{"istilah_karakter":"agent"}', 9),
('codm', 'Call of Duty Mobile', 'fps', 'fps', '{"istilah_karakter":"loadout dan operator"}', 10),

('efootball', 'eFootball', 'bola', 'bola', '{}', 11),
('easportsfc', 'EA Sports FC', 'bola', 'bola', '{}', 12),

('umum', 'Umum', 'umum', 'umum', '{}', 13);
