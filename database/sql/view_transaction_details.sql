SELECT
    `t`.`transaction_code`,
    `s`.`username` AS `sender_username`,
    `s`.`phone_no` AS `sender_phone`,
    `s`.`wallet_address` AS `sender_wallet_address`,
    `r`.`username` AS `receiver_username`,
    `r`.`phone_no` AS `receiver_phone`,
    `r`.`wallet_address` AS `receiver_wallet_address`,
    `t`.`amount`,
    `t`.`fee`,
    `t`.`status`,
    `t`.`timestamp`,
    `sig`.`signature`,
    `sig`.`public_key_hash`,
    `t`.`created_at`
FROM
    `transactions` `t`
    LEFT JOIN `wallet_details` `s` ON `t`.`sender` = `s`.`wallet_address`
    LEFT JOIN `wallet_details` `r` ON `t`.`receiver` = `r`.`wallet_address`
    LEFT JOIN `signatures` `sig` ON `sig`.`transaction_code` = `t`.`transaction_code`;