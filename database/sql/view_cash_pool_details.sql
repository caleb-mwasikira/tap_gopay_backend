--
-- Structure for view `cash_pool_details`
--
CREATE
OR REPLACE VIEW cash_pool_details AS
SELECT
    `uc`.`username` AS `creators_username`,
    `uc`.`email` AS `creators_email`,
    `p`.`pool_name`,
    `p`.`pool_type`,
    `p`.`description`,
    `p`.`wallet_address`,
    `p`.`target_amount`,
    `b`.`total_received` AS `collected_amount`,
    `p`.`expires_at`,
    `p`.`status`,
    -- Some cash pools eg. chama DO NOT have a predefined receiver.
    -- So receivers_* values could be NULL.
    -- COALESCE values into empty string
    COALESCE(`ur`.`username`, '') AS `receivers_username`,
    COALESCE(`ur`.`email`, '') AS `receivers_email`,
    COALESCE(`p`.`receiver`, '') AS `receivers_wallet_address`,
    `p`.`created_at`
FROM
    `cash_pools` AS `p`
    LEFT JOIN `wallet_owners` `wo` ON `wo`.`wallet_address` = `p`.`wallet_address`
    LEFT JOIN `users` `uc` ON `uc`.`id` = `wo`.`user_id`
    LEFT JOIN `balances` `b` ON `b`.`wallet_address` = `p`.`wallet_address`
    LEFT JOIN `wallet_owners` `rwo` ON `rwo`.`wallet_address` = `p`.`receiver`
    LEFT JOIN `users` `ur` ON `ur`.`id` = `rwo`.`user_id`;