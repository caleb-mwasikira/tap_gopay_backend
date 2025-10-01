--
-- Table structure for table `cash_pools`
--
CREATE TABLE
  `cash_pools` (
    `id` bigint NOT NULL,
    `pool_name` varchar(255) NOT NULL,
    `pool_type` ENUM('split_bill','chama','business_investment') NOT NULL,
    `description` text,
    `target_amount` decimal(10, 2) NOT NULL,
    `wallet_address` varchar(255) NOT NULL,
    `receiver` varchar(255) CHARACTER
    SET
      utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
      `expires_at` datetime NOT NULL,
      `status` enum (
        'open',
        'funded',
        'completed',
        'expired',
        'refunded'
      ) NOT NULL,
      `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

--
-- Triggers `cash_pools`
--
CREATE TRIGGER `verifyCashPool` BEFORE INSERT ON `cash_pools` FOR EACH ROW BEGIN
-- Check amount
IF NEW.target_amount < 100 THEN SIGNAL SQLSTATE '45000'
SET
  MESSAGE_TEXT = "Minimum target amount in a cash pool is KSH 100";

END IF;

-- Check expiry time
IF NEW.expires_at < NOW () THEN SIGNAL SQLSTATE '45000'
SET
  MESSAGE_TEXT = "Invalid cash pool expiry date time";

END IF;

END

--
-- AUTO_INCREMENT for table `cash_pools`
--
ALTER TABLE `cash_pools` MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

--
-- Constraints for table `cash_pools`
--
ALTER TABLE `cash_pools` ADD UNIQUE (`wallet_address`);
