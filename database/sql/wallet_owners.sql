--
-- Database: `tap_gopay`
--
-- --------------------------------------------------------
--
-- Table structure for table `wallet_owners`
--
CREATE TABLE `wallet_owners` (
    `id` bigint NOT NULL,
    `wallet_address` varchar(255) NOT NULL,
    `user_id` bigint NOT NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TRIGGER `restrictWalletOwners` BEFORE INSERT ON `wallet_owners`
FOR EACH ROW BEGIN
	DECLARE var_ownership_limit TINYINT;
    DECLARE var_total_owners TINYINT;

    --
    CALL walletExists(NEW.wallet_address, @wallet_exists, @wallet_active);
    IF NOT @wallet_exists THEN
        SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT="Referenced wallet does not exist";
    END IF;

    -- Fetch owner limit for wallet
    SELECT total_owners
    INTO var_ownership_limit
    FROM wallets
    WHERE wallet_address= NEW.wallet_address;

    -- Count existing number of wallet owners
    SELECT COUNT(*)
    INTO var_total_owners
    FROM wallet_owners
    WHERE wallet_address= NEW.wallet_address;

    -- Check if ownership limit is exceeded
    -- +1 Also includes NEW record
        IF (var_total_owners + 1) > var_ownership_limit THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="You've reached the maximum number of owners allowed for this wallet";
        END IF;

        IF var_total_owners = 0 THEN
            SET NEW.is_original_owner = 1;
        END IF;

END;