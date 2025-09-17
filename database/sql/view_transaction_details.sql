DROP TABLE IF EXISTS `transaction_details`;

DROP VIEW IF EXISTS `transaction_details`;

CREATE VIEW
    `transaction_details` AS
SELECT
    `t`.`transaction_id` AS `transaction_id`,
    `us`.`username` AS `senders_username`,
    `us`.`phone_no` AS `senders_phone`,
    `cs`.`wallet_address` AS `senders_wallet_address`,
    `ur`.`username` AS `receivers_username`,
    `ur`.`phone_no` AS `receivers_phone`,
    `cr`.`wallet_address` AS `receivers_wallet_address`,
    `t`.`amount` AS `amount`,
    `t`.`fee` AS `fee`,
    `t`.`timestamp` AS `timestamp`,
    `t`.`signature` AS `signature`,
    `t`.`public_key_hash` AS `public_key_hash`,
    `t`.`created_at` AS `created_at`
FROM
    (
        (
            (
                (
                    `transactions` `t`
                    join `wallets` `cs` on ((`t`.`sender` = `cs`.`wallet_address`))
                )
                join `users` `us` on ((`cs`.`user_id` = `us`.`id`))
            )
            join `wallets` `cr` on ((`t`.`receiver` = `cr`.`wallet_address`))
        )
        join `users` `ur` on ((`cr`.`user_id` = `ur`.`id`))
    );