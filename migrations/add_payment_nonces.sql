-- 创建 payment_nonces 表（防重放攻击）
-- 执行时间: 2026-03-03
-- 说明: 修复支付回调处理，防止重复处理同一笔支付

CREATE TABLE IF NOT EXISTS payment_nonces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id VARCHAR(100) NOT NULL,
    callback_type VARCHAR(50) NOT NULL,
    external_trade_no VARCHAR(100),
    processed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_nonces_transaction_id ON payment_nonces(transaction_id);
CREATE INDEX IF NOT EXISTS idx_payment_nonces_external_trade_no ON payment_nonces(external_trade_no);
CREATE INDEX IF NOT EXISTS idx_payment_nonces_expires_at ON payment_nonces(expires_at);

-- 验证表创建成功
SELECT 'payment_nonces 表创建成功，共 ' || COUNT(*) || ' 条记录' FROM payment_nonces;
