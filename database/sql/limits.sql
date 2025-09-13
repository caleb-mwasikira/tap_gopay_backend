-- --------------------------------------------------------
--
-- Table structure for table `limits`
--
CREATE TABLE `limits` (
  `id` int NOT NULL,
  `wallet_address` varchar(255) NOT NULL,
  `period` enum ('week', 'month', 'year') NOT NULL,
  `amount` decimal(10, 2) NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;