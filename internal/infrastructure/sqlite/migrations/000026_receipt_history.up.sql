-- Persist scanned receipts and editable purchase history.
CREATE TABLE IF NOT EXISTS receipt_history (
    id TEXT PRIMARY KEY,
    document_type TEXT NOT NULL DEFAULT 'receipt' CHECK (document_type IN ('receipt', 'invoice')),
    store_name TEXT NOT NULL CHECK (length(trim(store_name)) BETWEEN 1 AND 150),
    purchase_date TEXT NOT NULL,
    items_json TEXT NOT NULL DEFAULT '[]',
    total_amount REAL NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    confidence REAL NOT NULL DEFAULT 0 CHECK (confidence >= 0 AND confidence <= 1),
    expense_id TEXT,
    auto_add_to_finance INTEGER NOT NULL DEFAULT 0 CHECK (auto_add_to_finance IN (0, 1)),
    auto_update_pantry INTEGER NOT NULL DEFAULT 0 CHECK (auto_update_pantry IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_receipt_history_date
    ON receipt_history(purchase_date DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_receipt_history_store
    ON receipt_history(store_name COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_receipt_history_expense
    ON receipt_history(expense_id);