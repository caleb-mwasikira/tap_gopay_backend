
DROP TABLE IF EXISTS `public_keys`;
CREATE TABLE `public_keys` (
  `id` bigint NOT NULL,
  `email` varchar(255) NOT NULL,
  `public_key_hash` varchar(255) NOT NULL,
  `public_key` text NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
