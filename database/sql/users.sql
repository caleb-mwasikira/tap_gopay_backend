--
-- Table structure for table `users`
--
CREATE TABLE
  `users` (
    `id` bigint NOT NULL,
    `username` varchar(255) CHARACTER
    SET
      utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
      `email` varchar(255) CHARACTER
    SET
      utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,
      `password` varchar(255) NOT NULL,
      `phone_no` varchar(15) CHARACTER
    SET
      utf8mb4 COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,
      `email_verified` tinyint (1) NOT NULL DEFAULT '1',
      `role` enum ('user', 'admin', 'agent') NOT NULL DEFAULT 'user'
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

--
-- Indexes for table `users`
--
ALTER TABLE `users` ADD PRIMARY KEY (`id`),
ADD UNIQUE KEY `email` (`email`),
ADD UNIQUE KEY `phone_no` (`phone_no`);

--
-- AUTO_INCREMENT for table `users`
--
ALTER TABLE `users` MODIFY `id` bigint NOT NULL AUTO_INCREMENT,