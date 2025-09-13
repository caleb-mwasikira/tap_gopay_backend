-- --------------------------------------------------------
--
-- Structure for view `balances`
--
CREATE VIEW `balances` AS
SELECT
    `wallets`.`wallet_address` AS `wallet_address`,
    coalesce(
        sum(
            (
                case
                    when (`t`.`receiver` = `wallets`.`wallet_address`) then `t`.`amount`
                    else 0
                end
            )
        ),
        0
    ) AS `total_received`,
    coalesce(
        sum(
            (
                case
                    when (`t`.`sender` = `wallets`.`wallet_address`) then `t`.`amount`
                    else 0
                end
            )
        ),
        0
    ) AS `total_sent`,
    `wallet`.`initial_deposit` AS `initial_deposit`,
    (
        (
            `wallet`.`initial_deposit` + coalesce(
                sum(
                    (
                        case
                            when (`t`.`receiver` = `wallets`.`wallet_address`) then `t`.`amount`
                            else 0
                        end
                    )
                ),
                0
            )
        ) - coalesce(
            sum(
                (
                    case
                        when (`t`.`sender` = `wallets`.`wallet_address`) then `t`.`amount`
                        else 0
                    end
                )
            ),
            0
        )
    ) AS `balance`
FROM
    (
        (
            (
                select
                    distinct `wallets`.`wallet_address` AS `wallet_address`
                from
                    `wallets`
            ) `wallets`
            left join `transactions` `t` on(
                (
                    (`t`.`sender` = `wallets`.`wallet_address`)
                    or (`t`.`receiver` = `wallets`.`wallet_address`)
                )
            )
        )
        left join `wallets` `wallet` on(
            (
                `wallet`.`wallet_address` = `wallets`.`wallet_address`
            )
        )
    )
GROUP BY
    `wallets`.`wallet_address`,
    `wallet`.`initial_deposit`;