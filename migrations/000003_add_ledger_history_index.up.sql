CREATE INDEX idx_ledger_entries_history
    ON ledger_entries (account_id, created_at DESC, id DESC);