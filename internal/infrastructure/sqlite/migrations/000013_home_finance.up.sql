-- Migration 000013: Home Finance and Expenses Management
CREATE TABLE IF NOT EXISTS expenses (
    id                   TEXT PRIMARY KEY,
    title                TEXT NOT NULL,
    amount               REAL NOT NULL,
    currency             TEXT NOT NULL DEFAULT 'EUR',
    category             TEXT NOT NULL,
    subcategory          TEXT NOT NULL DEFAULT '',
    expense_date         TEXT NOT NULL,
    is_fixed             INTEGER NOT NULL DEFAULT 0,
    payment_method       TEXT NOT NULL DEFAULT 'card',
    payer_user_id        TEXT NOT NULL DEFAULT '',
    notes                TEXT NOT NULL DEFAULT '',
    recurring_frequency  TEXT NOT NULL DEFAULT 'none',
    contract_expiry_date TEXT,
    energy_kwh           REAL NOT NULL DEFAULT 0.0,
    gas_smc              REAL NOT NULL DEFAULT 0.0,
    food_waste_value     REAL NOT NULL DEFAULT 0.0,
    created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(expense_date);
CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses(category);
CREATE INDEX IF NOT EXISTS idx_expenses_is_fixed ON expenses(is_fixed);

CREATE TABLE IF NOT EXISTS budget_targets (
    id            TEXT PRIMARY KEY,
    category      TEXT NOT NULL,
    monthly_limit REAL NOT NULL,
    month_year    TEXT NOT NULL DEFAULT 'global',
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(category, month_year)
);
