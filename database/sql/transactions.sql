-- --------------------------------------------------------
--
-- Table structure for table `transactions`
--
CREATE TABLE `transactions` (
  `id` bigint NOT NULL,
  `transaction_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `sender` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `receiver` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `amount` decimal(10, 2) NOT NULL,
  `timestamp` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `signature` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `public_key_id` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

--
-- Triggers `transactions`.
-- This trigger is commented out as it requires SUPER_USER access
-- to execute.
-- TODO: Run trigger manually in database
--

-- DELIMITER $$

-- CREATE TRIGGER `verifyTransaction`
-- BEFORE INSERT ON `transactions`
-- FOR EACH ROW
-- BEGIN

--     -- Check if senders wallet exists and is active
--     IF NOT EXISTS(SELECT 1 FROM wallets WHERE wallet_address = NEW.sender AND is_active = TRUE) THEN
--     	SIGNAL SQLSTATE '45000'
--         SET MESSAGE_TEXT = "Senders wallet does NOT exist OR is NOT active";
--     END IF;

--     -- Check if receivers wallet exists and is active
--     IF NOT EXISTS(SELECT 1 FROM wallets WHERE wallet_address = NEW.receiver AND is_active = TRUE) THEN
--     	SIGNAL SQLSTATE '45000'
--         SET MESSAGE_TEXT = "Receivers wallet does NOT exist OR is NOT active";
--     END IF;

--     -- Check if valid amount
--     IF NEW.amount <= 0 THEN
--     	SIGNAL SQLSTATE '45000'
--         SET MESSAGE_TEXT = "Invalid transaction amount";
--     END IF;

--     -- Check if sender has enough balance to make transaction
--     SET @senders_balance = 0;
--     CALL getWalletBalance(NEW.sender, @senders_balance);

--     IF @senders_balance < NEW.amount THEN
--     	SIGNAL SQLSTATE '45000'
--         SET MESSAGE_TEXT = "Insufficient funds in senders wallet";
--     END IF;

-- END$$

-- DELIMITER ;