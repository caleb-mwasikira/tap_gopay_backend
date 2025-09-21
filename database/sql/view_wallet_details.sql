DROP TABLE IF EXISTS `wallet_details`;

DROP VIEW IF EXISTS `wallet_details`;

SELECT
    `u`.`id` AS `user_id`,
    `u`.`username` AS `username`,
    `u`.`phone_no` AS `phone_no`,
    `w`.`wallet_address` AS `wallet_address`,
    `w`.`wallet_name` AS `wallet_name`,
    `w`.`initial_deposit` AS `initial_deposit`,
    `w`.`is_active` AS `is_active`,
    `w`.`created_at` AS `created_at`,
    `b`.`balance` AS `balance`
FROM
    `tap_gopay`.`wallets` AS `w`
    JOIN `tap_gopay`.`wallet_owners` AS `wo` ON `wo`.`wallet_address` = `w`.`wallet_address`
    JOIN `tap_gopay`.`users` AS `u` ON `u`.`id` = `wo`.`user_id`
    JOIN `tap_gopay`.`balances` AS `b` ON `b`.`wallet_address` = `w`.`wallet_address`;