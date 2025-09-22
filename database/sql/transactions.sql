--
-- Table structure for table `transactions`
--

CREATE TABLE `transactions` (
  `id` bigint NOT NULL,
  `transaction_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `sender` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `receiver` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `fee` decimal(10,2) NOT NULL,
  `timestamp` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `signatures_count` tinyint NOT NULL DEFAULT '0',
  `status` enum('pending','confirmed','rejected') CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'pending',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TRIGGER IF EXISTS `verifyTransaction`;

CREATE TRIGGER `updateTransactionStatus` BEFORE UPDATE ON `transactions`
 FOR EACH ROW BEGIN
    DECLARE var_required_signatures TINYINT;

    -- Get required number of signatures
    SELECT required_signatures
    INTO var_required_signatures
    FROM wallets
    WHERE wallet_address = NEW.sender;

    -- Check if we have met the required number of signatures
    IF NEW.signatures_count >= var_required_signatures THEN
        SET NEW.status = 'confirmed';
    END IF;
END;

CREATE TRIGGER `verifyCashPoolTransactions` BEFORE INSERT ON `transactions`
 FOR EACH ROW BEGIN
	DECLARE var_withdrawing_cash_pool BOOLEAN;
    DECLARE var_depositing_cash_pool BOOLEAN;
    DECLARE var_target_amount DECIMAL(10,2);
    DECLARE var_collected_amount DECIMAL(10,2);
    DECLARE var_cash_pool_status TEXT;

    SET var_withdrawing_cash_pool = (NEW.sender LIKE '0xp00l%');
    SET var_depositing_cash_pool = (NEW.receiver LIKE '0xp00l%');

    IF var_withdrawing_cash_pool THEN
        -- Fetch cash pool target amount
        SELECT target_amount
        INTO var_target_amount
        FROM cash_pools WHERE wallet_address = NEW.sender;

        -- Fetch total amount collected into cash pool
        SELECT COALESCE(SUM(amount),0)
        INTO var_collected_amount
        FROM transactions
        WHERE receiver = NEW.sender;

        -- Restrict withdrawal from cash pool if target amount is not achieved
        IF var_collected_amount < var_target_amount THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="Withdrawals from cash pool restricted until target amount is achieved";
        END IF;

    ELSEIF var_depositing_cash_pool THEN
        -- Check if cash pool is still open
        SELECT status
        INTO var_cash_pool_status
        FROM cash_pools WHERE wallet_address = NEW.receiver;

        IF var_cash_pool_status <> "open" THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="Cash pool no longer open for depositing funds";
        END IF;

    END IF;

END;

CREATE TRIGGER `verifyTransaction` BEFORE INSERT ON `transactions`
FOR EACH ROW BEGIN
    DECLARE var_senders_balance DECIMAL(10,2);
    DECLARE var_amount DECIMAL(10,2);

    -- Add up transaction fees
    SET var_amount = NEW.amount + NEW.fee;

    -- Check if senders wallet exists and is active
    CALL isActiveWallet(NEW.sender, @sender_exists);

    IF NOT @sender_exists THEN
        SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Senders wallet does NOT exist OR is NOT active";
    END IF;

    -- Check if receivers wallet exists and is active
    CALL isActiveWallet(NEW.receiver, @receiver_exists);

    IF NOT @receiver_exists THEN
        SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Receivers wallet does NOT exist OR is NOT active";
    END IF;

    -- Check if valid amount
    IF var_amount < 1.0 THEN
        SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Minimum transferrable amount is KSH 1.0";
    END IF;

    -- Fetch senders balance
    CALL getWalletBalance(NEW.sender, @balance);
    SELECT @balance INTO var_senders_balance;

    -- Check amount < balance
    -- We add up transaction fees
    IF var_amount > var_senders_balance THEN
        SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Insufficient funds to complete transaction";
    END IF;

END;