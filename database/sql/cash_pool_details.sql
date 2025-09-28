--
-- Structure for view `cash_pool_details`
--
CREATE
OR REPLACE VIEW cash_pool_details AS
SELECT
    `uc`.`username` AS `creators_username`,
    `uc`.`email` AS `creators_email`,
    `ur`.`username` AS `receivers_username`,
    `ur`.`email` AS `receivers_email`,
    `p`.`pool_name` AS `pool_name`,
    `p`.`description`,
    `p`.`wallet_address`,
    `p`.`target_amount`,
    `b`.`total_received` AS `collected_amount`,
    `p`.`receiver` AS `receivers_wallet_address`,
    `p`.`expires_at`,
    `p`.`status`,
    `p`.`created_at`
FROM
    `cash_pools` AS `p`
    LEFT JOIN `wallet_owners` `wo` ON `wo`.`wallet_address` = `p`.`wallet_address`
    LEFT JOIN `users` `uc` ON `uc`.`id` = `wo`.`user_id`
    LEFT JOIN `balances` `b` ON `b`.`wallet_address` = `p`.`wallet_address`
    LEFT JOIN `wallet_owners` `rwo` ON `rwo`.`wallet_address` = `p`.`receiver`
    LEFT JOIN `users` `ur` ON `ur`.`id` = `rwo`.`user_id`;