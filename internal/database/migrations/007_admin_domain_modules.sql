CREATE TABLE IF NOT EXISTS store (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL DEFAULT 'Medusa Store',
 default_sales_channel_id TEXT,
 default_region_id TEXT,
 default_location_id TEXT,
 metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS store_currency (
 id TEXT PRIMARY KEY,
 store_id TEXT REFERENCES store(id) ON DELETE CASCADE,
 currency_code TEXT NOT NULL,
 is_default BOOLEAN NOT NULL DEFAULT false,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS store_locale (
 id TEXT PRIMARY KEY,
 store_id TEXT REFERENCES store(id) ON DELETE CASCADE,
 locale_code TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS "user" (
 id TEXT PRIMARY KEY,
 first_name TEXT,
 last_name TEXT,
 email TEXT NOT NULL,
 avatar_url TEXT,
 metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_user_email_unique ON "user"(email) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS invite (
 id TEXT PRIMARY KEY,
 email TEXT NOT NULL,
 accepted BOOLEAN NOT NULL DEFAULT false,
 token TEXT NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL,
 metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_invite_email_unique ON invite(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS IDX_invite_token ON invite(token) WHERE deleted_at IS NULL;
CREATE TABLE IF NOT EXISTS sales_channel (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 description TEXT,
 is_disabled BOOLEAN NOT NULL DEFAULT false,
 metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS region (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 currency_code TEXT NOT NULL,
 automatic_taxes BOOLEAN NOT NULL DEFAULT true,
 metadata JSONB,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 deleted_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS region_country (
 iso_2 TEXT PRIMARY KEY,
 iso_3 TEXT NOT NULL,
 num_code TEXT NOT NULL,
 name TEXT NOT NULL,
 display_name TEXT NOT NULL,
 region_id TEXT REFERENCES region(id) ON DELETE SET NULL,
 metadata JSONB
);
CREATE UNIQUE INDEX IF NOT EXISTS IDX_region_country_region_id_iso_2_unique ON region_country(region_id,iso_2);INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('af','afg','004','AFGHANISTAN','Afghanistan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('al','alb','008','ALBANIA','Albania') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('dz','dza','012','ALGERIA','Algeria') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('as','asm','016','AMERICAN SAMOA','American Samoa') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ad','and','020','ANDORRA','Andorra') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ao','ago','024','ANGOLA','Angola') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ai','aia','660','ANGUILLA','Anguilla') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('aq','ata','010','ANTARCTICA','Antarctica') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ag','atg','028','ANTIGUA AND BARBUDA','Antigua and Barbuda') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ar','arg','032','ARGENTINA','Argentina') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('am','arm','051','ARMENIA','Armenia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('aw','abw','533','ARUBA','Aruba') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('au','aus','036','AUSTRALIA','Australia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('at','aut','040','AUSTRIA','Austria') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('az','aze','031','AZERBAIJAN','Azerbaijan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bs','bhs','044','BAHAMAS','Bahamas') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bh','bhr','048','BAHRAIN','Bahrain') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bd','bgd','050','BANGLADESH','Bangladesh') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bb','brb','052','BARBADOS','Barbados') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('by','blr','112','BELARUS','Belarus') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('be','bel','056','BELGIUM','Belgium') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bz','blz','084','BELIZE','Belize') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bj','ben','204','BENIN','Benin') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bm','bmu','060','BERMUDA','Bermuda') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bt','btn','064','BHUTAN','Bhutan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bo','bol','068','BOLIVIA','Bolivia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bw','bwa','072','BOTSWANA','Botswana') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bv','bvd','074','BOUVET ISLAND','Bouvet Island') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('br','bra','076','BRAZIL','Brazil') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bn','brn','096','BRUNEI DARUSSALAM','Brunei Darussalam') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bg','bgr','100','BULGARIA','Bulgaria') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bf','bfa','854','BURKINA FASO','Burkina Faso') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bi','bdi','108','BURUNDI','Burundi') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('kh','khm','116','CAMBODIA','Cambodia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cm','cmr','120','CAMEROON','Cameroon') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ca','can','124','CANADA','Canada') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cv','cpv','132','CAPE VERDE','Cape Verde') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ky','cym','136','CAYMAN ISLANDS','Cayman Islands') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('td','tcd','148','CHAD','Chad') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cl','chl','152','CHILE','Chile') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cn','chn','156','CHINA','China') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cx','cxr','162','CHRISTMAS ISLAND','Christmas Island') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('co','col','170','COLOMBIA','Colombia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('km','com','174','COMOROS','Comoros') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cg','cog','178','CONGO','Congo') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ck','cok','184','COOK ISLANDS','Cook Islands') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cr','cri','188','COSTA RICA','Costa Rica') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ci','civ','384','COTE D''IVOIRE','Cote D''Ivoire') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('hr','hrv','191','CROATIA','Croatia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cu','cub','192','CUBA','Cuba') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cw','cuw','531','CURAÃ§AO','CuraÃ§ao') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cy','cyp','196','CYPRUS','Cyprus') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('cz','cze','203','CZECH REPUBLIC','Czech Republic') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('dk','dnk','208','DENMARK','Denmark') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('dj','dji','262','DJIBOUTI','Djibouti') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('dm','dma','212','DOMINICA','Dominica') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('do','dom','214','DOMINICAN REPUBLIC','Dominican Republic') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ec','ecu','218','ECUADOR','Ecuador') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('eg','egy','818','EGYPT','Egypt') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sv','slv','222','EL SALVADOR','El Salvador') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gq','gnq','226','EQUATORIAL GUINEA','Equatorial Guinea') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('er','eri','232','ERITREA','Eritrea') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ee','est','233','ESTONIA','Estonia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('et','eth','231','ETHIOPIA','Ethiopia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('fo','fro','234','FAROE ISLANDS','Faroe Islands') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('fj','fji','242','FIJI','Fiji') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('fi','fin','246','FINLAND','Finland') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('fr','fra','250','FRANCE','France') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gf','guf','254','FRENCH GUIANA','French Guiana') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pf','pyf','258','FRENCH POLYNESIA','French Polynesia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ga','gab','266','GABON','Gabon') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gm','gmb','270','GAMBIA','Gambia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ge','geo','268','GEORGIA','Georgia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('de','deu','276','GERMANY','Germany') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gh','gha','288','GHANA','Ghana') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gi','gib','292','GIBRALTAR','Gibraltar') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gr','grc','300','GREECE','Greece') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gl','grl','304','GREENLAND','Greenland') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gd','grd','308','GRENADA','Grenada') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gp','glp','312','GUADELOUPE','Guadeloupe') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gu','gum','316','GUAM','Guam') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gt','gtm','320','GUATEMALA','Guatemala') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gg','ggy','831','GUERNSEY','Guernsey') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gn','gin','324','GUINEA','Guinea') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gw','gnb','624','GUINEA-BISSAU','Guinea-Bissau') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gy','guy','328','GUYANA','Guyana') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ht','hti','332','HAITI','Haiti') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('hn','hnd','340','HONDURAS','Honduras') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('hk','hkg','344','HONG KONG','Hong Kong') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('hu','hun','348','HUNGARY','Hungary') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('is','isl','352','ICELAND','Iceland') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('in','ind','356','INDIA','India') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('id','idn','360','INDONESIA','Indonesia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('iq','irq','368','IRAQ','Iraq') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ie','irl','372','IRELAND','Ireland') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('im','imn','833','ISLE OF MAN','Isle Of Man') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('il','isr','376','ISRAEL','Israel') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('it','ita','380','ITALY','Italy') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('jm','jam','388','JAMAICA','Jamaica') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('jp','jpn','392','JAPAN','Japan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('je','jey','832','JERSEY','Jersey') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('jo','jor','400','JORDAN','Jordan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('kz','kaz','398','KAZAKHSTAN','Kazakhstan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ke','ken','404','KENYA','Kenya') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ki','kir','296','KIRIBATI','Kiribati') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('kr','kor','410','KOREA, REPUBLIC OF','Korea, Republic of') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('xk','xkx','900','KOSOVO','Kosovo') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('kw','kwt','414','KUWAIT','Kuwait') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('kg','kgz','417','KYRGYZSTAN','Kyrgyzstan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('lv','lva','428','LATVIA','Latvia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('lb','lbn','422','LEBANON','Lebanon') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ls','lso','426','LESOTHO','Lesotho') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('lr','lbr','430','LIBERIA','Liberia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ly','lby','434','LIBYA','Libya') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('li','lie','438','LIECHTENSTEIN','Liechtenstein') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('lt','ltu','440','LITHUANIA','Lithuania') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('lu','lux','442','LUXEMBOURG','Luxembourg') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mo','mac','446','MACAO','Macao') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mg','mdg','450','MADAGASCAR','Madagascar') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mw','mwi','454','MALAWI','Malawi') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('my','mys','458','MALAYSIA','Malaysia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mv','mdv','462','MALDIVES','Maldives') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ml','mli','466','MALI','Mali') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mt','mlt','470','MALTA','Malta') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mh','mhl','584','MARSHALL ISLANDS','Marshall Islands') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mq','mtq','474','MARTINIQUE','Martinique') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mr','mrt','478','MAURITANIA','Mauritania') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mu','mus','480','MAURITIUS','Mauritius') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('yt','myt','175','MAYOTTE','Mayotte') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mx','mex','484','MEXICO','Mexico') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('md','mda','498','MOLDOVA, REPUBLIC OF','Moldova, Republic of') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mc','mco','492','MONACO','Monaco') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mn','mng','496','MONGOLIA','Mongolia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('me','mne','499','MONTENEGRO','Montenegro') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ms','msr','500','MONTSERRAT','Montserrat') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ma','mar','504','MOROCCO','Morocco') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mz','moz','508','MOZAMBIQUE','Mozambique') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('mm','mmr','104','MYANMAR','Myanmar') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('na','nam','516','NAMIBIA','Namibia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('nr','nru','520','NAURU','Nauru') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('np','npl','524','NEPAL','Nepal') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('nl','nld','528','NETHERLANDS','Netherlands') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('nc','ncl','540','NEW CALEDONIA','New Caledonia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('nz','nzl','554','NEW ZEALAND','New Zealand') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ni','nic','558','NICARAGUA','Nicaragua') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ne','ner','562','NIGER','Niger') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ng','nga','566','NIGERIA','Nigeria') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('nu','niu','570','NIUE','Niue') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('nf','nfk','574','NORFOLK ISLAND','Norfolk Island') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('no','nor','578','NORWAY','Norway') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('om','omn','512','OMAN','Oman') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pk','pak','586','PAKISTAN','Pakistan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pw','plw','585','PALAU','Palau') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pa','pan','591','PANAMA','Panama') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pg','png','598','PAPUA NEW GUINEA','Papua New Guinea') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('py','pry','600','PARAGUAY','Paraguay') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pe','per','604','PERU','Peru') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ph','phl','608','PHILIPPINES','Philippines') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pn','pcn','612','PITCAIRN','Pitcairn') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pl','pol','616','POLAND','Poland') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pt','prt','620','PORTUGAL','Portugal') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('pr','pri','630','PUERTO RICO','Puerto Rico') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('qa','qat','634','QATAR','Qatar') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('re','reu','638','REUNION','Reunion') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ro','rom','642','ROMANIA','Romania') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ru','rus','643','RUSSIAN FEDERATION','Russian Federation') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('rw','rwa','646','RWANDA','Rwanda') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('bl','blm','652','SAINT BARTHÃ©LEMY','Saint BarthÃ©lemy') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sh','shn','654','SAINT HELENA','Saint Helena') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('lc','lca','662','SAINT LUCIA','Saint Lucia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ws','wsm','882','SAMOA','Samoa') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sm','smr','674','SAN MARINO','San Marino') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sa','sau','682','SAUDI ARABIA','Saudi Arabia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sn','sen','686','SENEGAL','Senegal') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('rs','srb','688','SERBIA','Serbia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sc','syc','690','SEYCHELLES','Seychelles') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sl','sle','694','SIERRA LEONE','Sierra Leone') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sg','sgp','702','SINGAPORE','Singapore') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sx','sxm','534','SINT MAARTEN','Sint Maarten') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sk','svk','703','SLOVAKIA','Slovakia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('si','svn','705','SLOVENIA','Slovenia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sb','slb','090','SOLOMON ISLANDS','Solomon Islands') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('so','som','706','SOMALIA','Somalia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('za','zaf','710','SOUTH AFRICA','South Africa') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ss','ssd','728','SOUTH SUDAN','South Sudan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('es','esp','724','SPAIN','Spain') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('lk','lka','144','SRI LANKA','Sri Lanka') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sd','sdn','729','SUDAN','Sudan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sr','sur','740','SURINAME','Suriname') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sz','swz','748','SWAZILAND','Swaziland') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('se','swe','752','SWEDEN','Sweden') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ch','che','756','SWITZERLAND','Switzerland') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('sy','syr','760','SYRIAN ARAB REPUBLIC','Syrian Arab Republic') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tj','tjk','762','TAJIKISTAN','Tajikistan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('th','tha','764','THAILAND','Thailand') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tl','tls','626','TIMOR LESTE','Timor Leste') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tg','tgo','768','TOGO','Togo') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tk','tkl','772','TOKELAU','Tokelau') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('to','ton','776','TONGA','Tonga') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tt','tto','780','TRINIDAD AND TOBAGO','Trinidad and Tobago') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tn','tun','788','TUNISIA','Tunisia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tr','tur','792','TURKEY','Turkey') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tm','tkm','795','TURKMENISTAN','Turkmenistan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('tv','tuv','798','TUVALU','Tuvalu') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ug','uga','800','UGANDA','Uganda') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ua','ukr','804','UKRAINE','Ukraine') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ae','are','784','UNITED ARAB EMIRATES','United Arab Emirates') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('gb','gbr','826','UNITED KINGDOM','United Kingdom') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('us','usa','840','UNITED STATES','United States') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('uy','ury','858','URUGUAY','Uruguay') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('uz','uzb','860','UZBEKISTAN','Uzbekistan') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('vu','vut','548','VANUATU','Vanuatu') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ve','ven','862','VENEZUELA','Venezuela') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('vn','vnm','704','VIET NAM','Viet Nam') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('vi','vir','850','VIRGIN ISLANDS, U.S.','Virgin Islands, U.S.') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('wf','wlf','876','WALLIS AND FUTUNA','Wallis and Futuna') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('eh','esh','732','WESTERN SAHARA','Western Sahara') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ye','yem','887','YEMEN','Yemen') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('zm','zmb','894','ZAMBIA','Zambia') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('zw','zwe','716','ZIMBABWE','Zimbabwe') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
INSERT INTO region_country(iso_2,iso_3,num_code,name,display_name) VALUES('ax','ala','248','Ã…LAND ISLANDS','Ã…land Islands') ON CONFLICT (iso_2) DO UPDATE SET iso_3=EXCLUDED.iso_3,num_code=EXCLUDED.num_code,name=EXCLUDED.name,display_name=EXCLUDED.display_name;
