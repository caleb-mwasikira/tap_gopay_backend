-- --------------------------------------------------------
--
-- Structure for view `wallet_details`
--
CREATE VIEW `wallet_details` AS
SELECT
    `u`.`id` AS `user_id`,
    `u`.`username` AS `username`,
    `u`.`phone_no` AS `phone_no`,
    `wallet`.`wallet_address` AS `wallet_address`,
    `wallet`.`initial_deposit` AS `initial_deposit`,
    `wallet`.`is_active` AS `is_active`,
    `wallet`.`created_at` AS `created_at`,
    `b`.`balance` AS `balance`
FROM
    (
        (
            `wallets` `wallet`
            join `users` `u` on((`u`.`id` = `wallet`.`user_id`))
        )
        join `balances` `b` on(
            (`b`.`wallet_address` = `wallet`.`wallet_address`)
        )
    );