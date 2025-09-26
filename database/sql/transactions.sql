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

CREATE TRIGGER `verifyCashPoolTransactions` BEFORE INSERT ON `transactions`
FOR EACH ROW BEGIN
    DECLARE is_withdrawing_from_cash_pool BOOLEAN;
    DECLARE is_depositing_into_cash_pool BOOLEAN;
    DECLARE var_target_amount DECIMAL(10,2);
    DECLARE var_collected_amount DECIMAL(10,2);
    DECLARE var_cash_pool_status TEXT;
    DECLARE var_cash_pool_balance DECIMAL(10,2);
    DECLARE var_cash_pool_receiver VARCHAR(255);

    SET is_withdrawing_from_cash_pool = (NEW.sender LIKE '33%');
    SET is_depositing_into_cash_pool = (NEW.receiver LIKE '33%');

    IF is_withdrawing_from_cash_pool THEN
        SELECT target_amount, receiver
        INTO var_target_amount, var_cash_pool_receiver
        FROM cash_pools WHERE wallet_address = NEW.sender;

        -- Check that funds being sent to correct receiver
        IF NEW.receiver != var_cash_pool_receiver THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="The recipient address you entered is invalid. Please check and try again.";
        END IF;

        SELECT balance
        INTO var_cash_pool_balance
        FROM balances
        WHERE wallet_address= NEW.sender;

        -- Check if cash pool has enough funds to withdraw
        IF (NEW.amount + NEW.fee) > var_cash_pool_balance THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="Insufficient funds in cash pool to complete transaction";
        END IF;

        SELECT COALESCE(SUM(amount),0)
        INTO var_collected_amount
        FROM transactions
        WHERE receiver = NEW.sender;

        -- Restrict withdrawal from cash pool if target amount is not achieved
        IF var_collected_amount < var_target_amount THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="Withdrawals from cash pool restricted until target amount is achieved";
        END IF;
    END IF;

    IF is_depositing_into_cash_pool THEN
        -- Check if cash pool is still open
        SELECT status
        INTO var_cash_pool_status
        FROM cash_pools WHERE wallet_address = NEW.receiver;

        IF var_cash_pool_status <> "open" THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="Cash pool no longer open for depositing funds";
        END IF;
    END IF;

END

CREATE TRIGGER `updateCashPool` BEFORE UPDATE ON `transactions`
FOR EACH ROW BEGIN
    DECLARE var_target_amount DECIMAL(10,2);
    DECLARE var_collected_amount DECIMAL(10,2);
    DECLARE is_depositing_into_cash_pool TINYINT(1);

    -- Check if receiver is a cash pool
    SET is_depositing_into_cash_pool = (NEW.receiver LIKE '33%');

    IF NEW.status = 'confirmed' AND is_depositing_into_cash_pool THEN
        -- Add to collected amount
        UPDATE cash_pools
        SET collected_amount = collected_amount + NEW.amount
        WHERE wallet_address = NEW.receiver;

        -- Fetch target and collected amount
        SELECT target_amount, collected_amount
        INTO var_target_amount, var_collected_amount
        FROM cash_pools
        WHERE wallet_address = NEW.receiver;

        -- Check if target reached
        IF var_collected_amount >= var_target_amount THEN
            UPDATE cash_pools
            SET status = 'funded'
            WHERE wallet_address = NEW.receiver;
        END IF;
    END IF;

END;