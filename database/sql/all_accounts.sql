CREATE
OR REPLACE VIEW all_accounts AS
SELECT
    `w`.`wallet_address`,
    `u`.`username`,
    `u`.`phone_no`,
    'wallet' AS `account_type`
FROM
    `wallets` `w`
    JOIN `wallet_owners` `wo` ON `wo`.`wallet_address` = `w`.`wallet_address`
    JOIN `users` `u` ON `u`.`id` = `wo`.`user_id`
UNION ALL
SELECT
    `cp`.`wallet_address`,
    `u`.`username`,
    `u`.`phone_no`,
    'cash_pool' AS `account_type`
FROM
    `cash_pools` `cp`
    JOIN `users` `u` ON `u`.`id` = `cp`.`creator_user_id`;