--
-- Database: `tap_gopay`
--
-- --------------------------------------------------------
--
-- Table structure for table `signatures`
--
CREATE TABLE
  `signatures` (
    `id` bigint NOT NULL,
    `transaction_code` bigint NOT NULL,
    `user_id` bigint NOT NULL,
    `signature` text NOT NULL,
    `public_key_hash` varchar(255) NOT NULL
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;