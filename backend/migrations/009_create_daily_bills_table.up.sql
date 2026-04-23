CREATE TABLE IF NOT EXISTS daily_bills (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    bill_date DATE NOT NULL,
    total_journeys INTEGER DEFAULT 0,
    total_distance DECIMAL(10, 2) DEFAULT 0,
    total_fare DECIMAL(10, 2) DEFAULT 0,
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'paid', 'failed')),
    payment_id VARCHAR(255),
    payment_method VARCHAR(50),
    paid_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, bill_date)
);

CREATE INDEX idx_daily_bills_user_id ON daily_bills(user_id);
CREATE INDEX idx_daily_bills_bill_date ON daily_bills(bill_date);
CREATE INDEX idx_daily_bills_status ON daily_bills(status);
CREATE INDEX idx_daily_bills_user_date ON daily_bills(user_id, bill_date);


