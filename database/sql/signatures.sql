--
-- Table structure for table `signatures`
--

CREATE TABLE `signatures` (
  `id` bigint NOT NULL,
  `user_id` bigint NOT NULL,
  `transaction_id` bigint NOT NULL,
  `transaction_code` varchar(25) NOT NULL,
  `signature` text NOT NULL,
  `public_key_hash` varchar(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

--
-- Triggers `signatures`
--
CREATE TRIGGER `updateSignatureCount` AFTER INSERT ON `signatures` FOR EACH ROW BEGIN
UPDATE transactions
SET
  signatures_count = signatures_count + 1
WHERE
  transaction_code = NEW.transaction_code;

END;

CREATE TRIGGER `verifySignature` BEFORE INSERT ON `signatures`
 FOR EACH ROW BEGIN
    DECLARE var_sender VARCHAR(255);
    DECLARE var_transaction_type VARCHAR(255);
    DECLARE var_transaction_id BIGINT;

    -- Get sender's wallet address
    SELECT sender, transaction_type
    INTO var_sender, var_transaction_type
    FROM transactions
    WHERE id = NEW.transaction_id;

    IF var_transaction_type <> 'refund' THEN
        -- Check ownership
        IF NOT EXISTS (
            SELECT 1
            FROM wallet_owners
            WHERE wallet_address = var_sender
              AND user_id = NEW.user_id
        ) THEN
            SIGNAL SQLSTATE '45000'
            SET MESSAGE_TEXT = 'This wallet does not belong to you';
        END IF;
    END IF;

END;

--
-- Indexes for table `signatures`
--
ALTER TABLE `signatures`
  ADD PRIMARY KEY (`id`),
  ADD KEY `fk_signatures_user_id` (`user_id`),
  ADD KEY `fk_signatures_transaction_id` (`transaction_id`);

--
-- AUTO_INCREMENT for table `signatures`
--
ALTER TABLE `signatures`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT, AUTO_INCREMENT=4;

--
-- Constraints for table `signatures`
--
ALTER TABLE `signatures`
  ADD CONSTRAINT `fk_signatures_transaction_id` FOREIGN KEY (`transaction_id`) REFERENCES `transactions` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `fk_signatures_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);

