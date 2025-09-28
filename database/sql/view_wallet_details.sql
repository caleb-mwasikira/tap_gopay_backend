DROP TABLE IF EXISTS `wallet_details`;

DROP VIEW IF EXISTS `wallet_details`;

SELECT
    u.id AS user_id,
    u.username AS username,
    u.phone_no AS phone_no,
    w.wallet_address AS wallet_address,
    w.wallet_name AS wallet_name,
    w.initial_deposit AS initial_deposit,
    w.is_active AS is_active,
    w.created_at AS created_at,
    b.balance AS balance
FROM (
    SELECT wallet_address, wallet_name, initial_deposit, is_active, created_at
    FROM wallets
    UNION ALL
    SELECT wallet_address, pool_name AS wallet_name, 0.0 AS initial_deposit,
           (expires_at > NOW()) AS is_active, created_at
    FROM cash_pools
) AS w
JOIN wallet_owners wo ON wo.wallet_address = w.wallet_address
JOIN users u ON u.id = wo.user_id
LEFT JOIN balances b ON b.wallet_address = w.wallet_address;
