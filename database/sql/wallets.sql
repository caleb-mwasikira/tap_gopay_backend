--
-- Database: `tap_gopay`
--
-- --------------------------------------------------------
--
-- Table structure for table `wallets`
--
CREATE TABLE
  `wallets` (
    `id` bigint NOT NULL,
    `wallet_address` varchar(255) CHARACTER
    SET
      utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
      `wallet_name` varchar(100) CHARACTER
    SET
      utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
      `initial_deposit` decimal(10, 2) NOT NULL DEFAULT '0.00',
      `is_active` tinyint NOT NULL DEFAULT '1',
      `total_owners` tinyint NOT NULL DEFAULT '1',
      `required_signatures` tinyint NOT NULL DEFAULT '1',
      `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TRIGGER `verifyWallet` BEFORE INSERT ON `wallets`
FOR EACH ROW BEGIN

  -- Check required_signatures is NOT gt total_owners
  IF NEW.required_signatures > NEW.total_owners THEN
    SIGNAL SQLSTATE '45000'
    SET MESSAGE_TEXT="Number of signatures required cannot exceed total number of owners";
  END IF;

END;
