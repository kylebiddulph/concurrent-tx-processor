CREATE TABLE transactions (
    transaction_id VARCHAR(255) PRIMARY KEY,
    amount DECIMAL(20, 2) NOT NULL,
    status ENUM('PENDING', 'COMPLETED', 'FAILED') NOT NULL DEFAULT 'PENDING',
    date_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    date_created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Transaction locks table for managing concurrent access
CREATE TABLE transaction_locks (
transaction_id VARCHAR(255) PRIMARY KEY,
locked TINYINT(1) NOT NULL DEFAULT 0,
version BIGINT NOT NULL DEFAULT 0,
date_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
date_created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
FOREIGN KEY (transaction_id) REFERENCES transactions(transaction_id)
);