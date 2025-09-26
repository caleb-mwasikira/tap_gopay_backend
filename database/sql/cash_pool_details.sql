--
-- Structure for view `cash_pool_details`
--
CREATE
OR REPLACE VIEW cash_pool_details AS
SELECT
    `uc`.`username` AS `creators_username`,
    `uc`.`email` AS `creators_email`,
    `p`.`pool_name` AS `pool_name`,
    `p`.`description`,
    `p`.`wallet_address`,
    `p`.`target_amount`,
    `ur`.`username` AS `receivers_username`,
    `ur`.`email` AS `receivers_email`,
    `p`.`receiver` AS `receivers_wallet_address`,
    `p`.`expires_at`,
    `p`.`status`,
    `p`.`collected_amount`,
    `p`.`created_at`
FROM
    `cash_pools` AS `p`
    JOIN `users` `uc` ON `uc`.`id` = `p`.`creator_user_id`
    JOIN `wallet_owners` `wo` ON `wo`.`wallet_address` = `p`.`receiver`
    JOIN `users` `ur` ON `ur`.`id` = `wo`.`user_id`;