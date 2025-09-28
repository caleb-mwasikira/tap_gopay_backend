--
-- Table structure for table `transactions`
--
CREATE TABLE `transactions` (
  `id` bigint NOT NULL,
  `transaction_code` varchar(25) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `refund_transaction_code` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
  `sender` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `receiver` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `fee` decimal(10,2) NOT NULL,
  `timestamp` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `signatures_count` tinyint NOT NULL DEFAULT '0',
  `status` enum('pending','confirmed','rejected') CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'pending',
  `transaction_type` enum('transfer','refund') CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL DEFAULT 'transfer',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


CREATE TRIGGER `updateTransactionStatus` BEFORE UPDATE ON `transactions`
FOR EACH ROW
BEGIN
    DECLARE var_required_signatures TINYINT;

    IF NEW.transaction_type= 'transfer' THEN
        -- Get required number of signatures
        SELECT required_signatures
        INTO var_required_signatures
        FROM wallets
        WHERE wallet_address = NEW.sender;

        -- Check if we have met the required number of signatures
        IF NEW.signatures_count >= var_required_signatures THEN
            SET NEW.status = 'confirmed';
        END IF;
    ELSE
        SET NEW.status = 'confirmed';
    END IF;
END;

CREATE TRIGGER `verifyTransaction`
BEFORE INSERT ON `transactions`
FOR EACH ROW
BEGIN
    DECLARE var_senders_balance DECIMAL(10,2);
    DECLARE var_amount DECIMAL(10,2);
    DECLARE var_transaction_fee DECIMAL(10,2) DEFAULT 0;
    DECLARE var_exists BOOLEAN DEFAULT FALSE;

    DECLARE sender_exists BOOLEAN DEFAULT FALSE;
    DECLARE sender_active BOOLEAN DEFAULT FALSE;
    DECLARE receiver_exists BOOLEAN DEFAULT FALSE;
    DECLARE receiver_active BOOLEAN DEFAULT FALSE;

    IF NEW.transaction_type = 'refund' THEN
        -- Drop transaction fees in refunds
        SET NEW.fee = 0.0;

        -- Check if refund transaction code present
        IF NEW.refund_transaction_code IS NULL THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Refund failed: missing refund transaction code';
        END IF;

        -- Check if the referenced transaction exists
        SELECT EXISTS(
          SELECT 1 FROM transactions WHERE transaction_code = NEW.refund_transaction_code
        ) INTO var_exists;

        IF var_exists = 0 THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = "We couldn't find the transaction you are trying to refund. Please check transaction code and try again";
        END IF;

        -- Total amount including fee (fee=0 here)
        SET var_amount = NEW.amount + NEW.fee;

        -- Only check wallets exist (ignore active status)
        CALL walletExists(NEW.sender, @sender_exists, @sender_active);
        SELECT @sender_exists, @sender_active INTO sender_exists, sender_active;
        IF NOT sender_exists THEN
            SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT = 'Sender wallet does NOT exist';
        END IF;

        CALL walletExists(NEW.receiver, @receiver_exists, @receiver_active);
        SELECT @receiver_exists, @receiver_active INTO receiver_exists, receiver_active;
        IF NOT receiver_exists THEN
            SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT = 'Receiver wallet does NOT exist';
        END IF;

    ELSEIF NEW.transaction_type = 'transfer' THEN
        -- Verify transaction fees
        SELECT COALESCE(fee, 0)
        INTO var_transaction_fee
        FROM transaction_fees
        WHERE NEW.amount BETWEEN min_amount AND max_amount
        LIMIT 1;

        IF NEW.fee <> var_transaction_fee THEN
            SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT = 'Invalid transaction fee';
        END IF;

        -- Total amount including fee
        SET var_amount = NEW.amount + NEW.fee;

        -- Check sender wallet exists AND active
        CALL walletExists(NEW.sender, @sender_exists, @sender_active);
        SELECT @sender_exists, @sender_active INTO sender_exists, sender_active;
        IF NOT sender_exists OR NOT sender_active THEN
            SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT = 'Sender wallet does NOT exist OR is NOT active';
        END IF;

        -- Check receiver wallet exists AND active
        CALL walletExists(NEW.receiver, @receiver_exists, @receiver_active);
        SELECT @receiver_exists, @receiver_active INTO receiver_exists, receiver_active;
        IF NOT receiver_exists OR NOT receiver_active THEN
            SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT = 'Receiver wallet does NOT exist OR is NOT active';
        END IF;
    END IF;

    IF var_amount < 1.0 THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Minimum transferable amount is KSH 1.0';
    END IF;

    -- Fetch sender's balance
    CALL getWalletBalance(NEW.sender, @balance);
    SELECT @balance INTO var_senders_balance;

    -- Ensure sender has enough funds (including fees)
    IF var_amount > var_senders_balance THEN
        SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'Insufficient funds to complete transaction';
    END IF;
END


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
        SELECT target_amount, receivers_wallet_address, collected_amount
        INTO var_target_amount, var_cash_pool_receiver, var_collected_amount
        FROM cash_pool_details WHERE wallet_address = NEW.sender;

        IF NEW.transaction_type= 'transfer' THEN
            -- Check that funds are being sent to correct receiver
            IF var_cash_pool_receiver != NEW.receiver THEN
                SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT="Receiver specified does not match the expected recipient for this payment";
            END IF;

            -- Restrict withdrawal from cash pool if target amount is not achieved
            IF var_collected_amount < var_target_amount THEN
                SIGNAL SQLSTATE '45000'
                SET MESSAGE_TEXT="Withdrawals from cash pool restricted until target amount is achieved";
            END IF;
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
    END IF;

    IF is_depositing_into_cash_pool THEN
        SELECT status, target_amount, collected_amount
        INTO var_cash_pool_status, var_target_amount, var_collected_amount
        FROM cash_pool_details WHERE wallet_address = NEW.receiver;

        IF var_cash_pool_status <> "open" THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="Cash pool no longer open for depositing funds";
        END IF;

        -- Check if target amount being exceeded
        IF var_collected_amount >= var_target_amount THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT="Cash pool has already reached its funding goal";
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
        -- Fetch target and collected amount
        SELECT target_amount, collected_amount
        INTO var_target_amount, var_collected_amount
        FROM cash_pool_details
        WHERE wallet_address = NEW.receiver;

        -- Check if target reached
        IF var_collected_amount >= var_target_amount THEN
            UPDATE cash_pools
            SET status = 'funded'
            WHERE wallet_address = NEW.receiver;
        END IF;
    END IF;

END;