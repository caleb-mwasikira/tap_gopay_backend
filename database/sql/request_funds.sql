DROP TABLE IF EXISTS `request_funds`;

CREATE TABLE
  `request_funds` (
    `id` int NOT NULL,
    `transaction_code` varchar(25) CHARACTER
    SET
      utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
      `sender` varchar(255) NOT NULL,
      `receiver` varchar(255) NOT NULL,
      `amount` decimal(10, 2) NOT NULL,
      `timestamp` varchar(30) CHARACTER
    SET
      utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
      -- Only one signature is required, which belongs to
      -- the user requesting funds
      -- So no need of placing signature in different table
      `signature` varchar(255) NOT NULL,
      `public_key_hash` varchar(255) NOT NULL,
      `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;