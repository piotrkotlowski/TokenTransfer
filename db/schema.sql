
CREATE TABLE IF NOT EXISTS wallets (
    address TEXT PRIMARY KEY,
    balance NUMERIC NOT NULL
);

CREATE TABLE IF NOT EXISTS transactionhistory (
    transaction_id BIGSERIAL PRIMARY KEY,
    sender   TEXT NOT NULL,
    receiver TEXT NOT NULL,
    amount   NUMERIC NOT NULL,
    CONSTRAINT fk_sender
      FOREIGN KEY(sender)
        REFERENCES wallets(address), 
    CONSTRAINT fk_receiver
      FOREIGN KEY(receiver)
        REFERENCES wallets(address)   
);

INSERT INTO wallets(address, balance)
VALUES ('0x0000000000000000000000000000000000000000', 1000000)
ON CONFLICT DO NOTHING;