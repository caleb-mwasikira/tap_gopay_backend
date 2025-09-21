--
-- Database: `tap_gopay`
--
-- --------------------------------------------------------
--
-- Table structure for table `wallet_owners`
--
CREATE TABLE
  `wallet_owners` (
    `id` bigint NOT NULL,
    `wallet_address` varchar(255) NOT NULL,
    `user_id` bigint NOT NULL
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;