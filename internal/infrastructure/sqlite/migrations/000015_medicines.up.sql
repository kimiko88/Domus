-- Migration 000015: Medicine Cabinet & Expiration Tracking
CREATE TABLE IF NOT EXISTS medicines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    active_ingredient TEXT,
    dosage TEXT,
    form TEXT,
    aic TEXT,
    barcode TEXT,
    lot_number TEXT,
    expiration_date TEXT NOT NULL,
    pao_days INTEGER NOT NULL DEFAULT 0,
    opened_at TEXT,
    effective_expiration_date TEXT NOT NULL,
    location TEXT NOT NULL DEFAULT 'armadietto_bagno',
    requires_cold_chain INTEGER NOT NULL DEFAULT 0,
    quantity REAL NOT NULL DEFAULT 1.0,
    unit TEXT NOT NULL DEFAULT 'compresse',
    min_threshold REAL NOT NULL DEFAULT 0.0,
    prescription_required INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_medicines_effective_exp ON medicines(effective_expiration_date);
CREATE INDEX IF NOT EXISTS idx_medicines_aic ON medicines(aic);
CREATE INDEX IF NOT EXISTS idx_medicines_barcode ON medicines(barcode);
CREATE INDEX IF NOT EXISTS idx_medicines_location ON medicines(location);
CREATE INDEX IF NOT EXISTS idx_medicines_cold_chain ON medicines(requires_cold_chain);
