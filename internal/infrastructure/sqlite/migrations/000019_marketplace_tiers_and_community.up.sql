-- Migration 000019: Marketplace Tiers, Verification and Community Open Source Packages
ALTER TABLE store_packages ADD COLUMN tier TEXT NOT NULL DEFAULT 'verified';
ALTER TABLE store_packages ADD COLUMN review_status TEXT NOT NULL DEFAULT 'approved';
ALTER TABLE store_packages ADD COLUMN repository_url TEXT DEFAULT '';
ALTER TABLE store_packages ADD COLUMN license TEXT DEFAULT 'Apache-2.0';
ALTER TABLE store_packages ADD COLUMN capabilities_json TEXT DEFAULT '[]';
ALTER TABLE store_packages ADD COLUMN icon TEXT DEFAULT 'box';
