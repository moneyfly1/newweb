-- 支付回调防重放 nonce 表
CREATE TABLE IF NOT EXISTS payment_nonces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id VARCHAR(100) NOT NULL UNIQUE,
    callback_type VARCHAR(50) NOT NULL,
    external_trade_no VARCHAR(100),
    processed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

CREATE INDEX idx_payment_nonces_expires ON payment_nonces(expires_at);
CREATE INDEX idx_payment_nonces_lookup ON payment_nonces(transaction_id, callback_type);
