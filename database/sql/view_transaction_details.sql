DROP TABLE IF EXISTS `transaction_details`;

DROP VIEW IF EXISTS `transaction_details`;

SELECT
    `t`.`transaction_code` AS `transaction_code`,
    `us`.`username` AS `senders_username`,
    `us`.`phone_no` AS `senders_phone`,
    `sw`.`wallet_address` AS `senders_wallet_address`,
    `ur`.`username` AS `receivers_username`,
    `ur`.`phone_no` AS `receivers_phone`,
    `rw`.`wallet_address` AS `receivers_wallet_address`,
    `t`.`amount` AS `amount`,
    `t`.`fee` AS `fee`,
    `t`.`timestamp` AS `timestamp`,
    `sig`.`user_id` AS `signer_user_id`,
    `u`.`username` AS `signer_username`,
    `sig`.`signature` AS `signature`,
    `sig`.`public_key_hash` AS `public_key_hash`,
    `t`.`created_at` AS `created_at`
FROM
    `tap_gopay`.`transactions` AS `t`
    -- sender wallet
    JOIN `tap_gopay`.`wallets` AS `sw` ON `t`.`sender` = `sw`.`wallet_address`
    JOIN `tap_gopay`.`wallet_owners` AS `wo_s` ON `wo_s`.`wallet_address` = `sw`.`wallet_address`
    JOIN `tap_gopay`.`users` AS `us` ON `us`.`id` = `wo_s`.`user_id`
    -- receiver wallet
    JOIN `tap_gopay`.`wallets` AS `rw` ON `t`.`receiver` = `rw`.`wallet_address`
    JOIN `tap_gopay`.`wallet_owners` AS `wo_r` ON `wo_r`.`wallet_address` = `rw`.`wallet_address`
    JOIN `tap_gopay`.`users` AS `ur` ON `ur`.`id` = `wo_r`.`user_id`
    -- signatures
    LEFT JOIN `tap_gopay`.`signatures` AS `sig` ON `sig`.`transaction_code` = `t`.`transaction_code`
    LEFT JOIN `tap_gopay`.`users` AS `u` ON `u`.`id` = `sig`.`user_id`;