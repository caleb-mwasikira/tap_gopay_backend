
DROP TABLE IF EXISTS `transactions`;

CREATE TABLE `transactions` (
  `id` bigint NOT NULL,
  `transaction_id` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `sender` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `receiver` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `amount` decimal(10,2) NOT NULL,
  `fee` decimal(10,2) NOT NULL,
  `timestamp` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `signature` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `public_key_hash` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

DROP TRIGGER IF EXISTS `verifyTransaction`;

CREATE TRIGGER `verifyTransaction` 
BEFORE INSERT ON `transactions` 
FOR EACH ROW 
BEGIN
    DECLARE var_senders_balance DECIMAL(10,2);

    -- Check if senders wallet exists and is active
    IF NOT EXISTS(SELECT 1 FROM wallets WHERE wallet_address = NEW.sender AND is_active = TRUE) THEN
    	SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Senders wallet does NOT exist OR is NOT active";
    END IF;

    -- Check if receivers wallet exists and is active
    IF NOT EXISTS(SELECT 1 FROM wallets WHERE wallet_address = NEW.receiver AND is_active = TRUE) THEN
    	SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Receivers wallet does NOT exist OR is NOT active";
    END IF;

    -- Check if valid amount
    IF NEW.amount < 1.0 THEN
    	SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Minimum transferrable amount is KSH 1.0";
    END IF;

    -- Fetch senders balance
    CALL getWalletBalance(NEW.sender, @balance);
    SELECT @balance INTO var_senders_balance;

    -- Check amount < balance
    -- At this point we assume amount= actual amount + transaction fees.
    -- So there is no need of adding up transaction fees at this point.
    IF NEW.amount > var_senders_balance THEN
      SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Insufficient funds to complete transaction";
    END IF;

END;

