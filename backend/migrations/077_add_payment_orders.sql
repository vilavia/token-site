CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    order_no VARCHAR(64) NOT NULL UNIQUE,
    trade_no VARCHAR(64),
    amount_usd DECIMAL(20,8) NOT NULL,
    amount_rmb DECIMAL(20,2) NOT NULL,
    exchange_rate DECIMAL(10,4) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    pay_type VARCHAR(32),
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payment_orders_user_id ON payment_orders(user_id);
CREATE INDEX idx_payment_orders_status ON payment_orders(status);
