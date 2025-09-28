DROP TABLE IF EXISTS `balances`;
DROP VIEW IF EXISTS `balances`;

CREATE OR REPLACE VIEW `balances` AS
SELECT
    w.wallet_address AS wallet_address,

    -- Total received by this wallet
    COALESCE(
        SUM(
            CASE
                WHEN t.receiver = w.wallet_address THEN t.amount
                ELSE 0
            END
        ),
        0
    ) AS total_received,

    -- Total sent by this wallet
    COALESCE(
        SUM(
            CASE
                WHEN t.sender = w.wallet_address THEN t.amount
                ELSE 0
            END
        ),
        0
    ) AS total_sent,

    -- Initial deposit
    COALESCE(wl.initial_deposit, 0.0) AS initial_deposit,

    -- Current balance calculation
    (
        COALESCE(wl.initial_deposit, 0)
        + COALESCE(
            SUM(
                CASE
                    WHEN t.receiver = w.wallet_address THEN t.amount
                    ELSE 0
                END
            ),
            0
        )
        - COALESCE(
            SUM(
                CASE
                    WHEN t.sender = w.wallet_address THEN t.amount
                    ELSE 0
                END
            ),
            0
        )
        - COALESCE(
            SUM(
                CASE
                    WHEN t.sender = w.wallet_address THEN t.fee
                    ELSE 0
                END
            ),
            0
        )
    ) AS balance

FROM (
    SELECT DISTINCT wallet_address FROM wallets
    UNION
    SELECT DISTINCT wallet_address FROM cash_pools
) w
LEFT JOIN transactions t
    ON (t.sender = w.wallet_address OR t.receiver = w.wallet_address)
    AND t.status = 'confirmed'
LEFT JOIN wallets wl
    ON wl.wallet_address = w.wallet_address
GROUP BY
    w.wallet_address;

