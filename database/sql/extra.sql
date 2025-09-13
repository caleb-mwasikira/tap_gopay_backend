--
-- Indexes for dumped tables
--
--
-- Indexes for table `wallets`
--
ALTER TABLE
  `wallets`
ADD
  PRIMARY KEY (`id`),
ADD
  UNIQUE KEY `wallet_address` (`wallet_address`),
ADD
  KEY `user_id` (`user_id`);

--
-- Indexes for table `limits`
--
ALTER TABLE
  `limits`
ADD
  PRIMARY KEY (`id`),
ADD
  UNIQUE KEY `wallet_address` (`wallet_address`);

--
-- Indexes for table `password_reset_tokens`
--
ALTER TABLE
  `password_reset_tokens`
ADD
  PRIMARY KEY (`id`);

--
-- Indexes for table `request_funds`
--
ALTER TABLE
  `request_funds`
ADD
  PRIMARY KEY (`id`),
ADD
  KEY `sender` (`sender`),
ADD
  KEY `receiver` (`receiver`);

--
-- Indexes for table `transactions`
--
ALTER TABLE
  `transactions`
ADD
  PRIMARY KEY (`id`),
ADD
  KEY `fk_transactions_receiver` (`receiver`),
ADD
  KEY `fk_transactions_sender` (`sender`);

--
-- Indexes for table `users`
--
ALTER TABLE
  `users`
ADD
  PRIMARY KEY (`id`),
ADD
  UNIQUE KEY `email` (`email`),
ADD
  UNIQUE KEY `username` (`username`),
ADD
  UNIQUE KEY `email_2` (`email`, `phone_no`);

--
-- AUTO_INCREMENT for dumped tables
--
--
-- AUTO_INCREMENT for table `wallets`
--
ALTER TABLE
  `wallets`
MODIFY
  `id` bigint NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `limits`
--
ALTER TABLE
  `limits`
MODIFY
  `id` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `password_reset_tokens`
--
ALTER TABLE
  `password_reset_tokens`
MODIFY
  `id` bigint NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `request_funds`
--
ALTER TABLE
  `request_funds`
MODIFY
  `id` int NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `transactions`
--
ALTER TABLE
  `transactions`
MODIFY
  `id` bigint NOT NULL AUTO_INCREMENT;

--
-- AUTO_INCREMENT for table `users`
--
ALTER TABLE
  `users`
MODIFY
  `id` bigint NOT NULL AUTO_INCREMENT;

--
-- Constraints for dumped tables
--
--
-- Constraints for table `wallets`
--
ALTER TABLE
  `wallets`
ADD
  CONSTRAINT `wallets_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);

--
-- Constraints for table `limits`
--
ALTER TABLE
  `limits`
ADD
  CONSTRAINT `fk_limits_wallet_address` FOREIGN KEY (`wallet_address`) REFERENCES `wallets` (`wallet_address`) ON DELETE CASCADE ON UPDATE CASCADE;

--
-- Constraints for table `request_funds`
--
ALTER TABLE
  `request_funds`
ADD
  CONSTRAINT `fk_request_funds_receiver` FOREIGN KEY (`receiver`) REFERENCES `wallets` (`wallet_address`) ON DELETE RESTRICT ON UPDATE RESTRICT,
ADD
  CONSTRAINT `fk_request_funds_sender` FOREIGN KEY (`sender`) REFERENCES `wallets` (`wallet_address`) ON DELETE RESTRICT ON UPDATE RESTRICT;

--
-- Constraints for table `transactions`
--
ALTER TABLE
  `transactions`
ADD
  CONSTRAINT `fk_transactions_receiver` FOREIGN KEY (`receiver`) REFERENCES `wallets` (`wallet_address`) ON DELETE RESTRICT ON UPDATE RESTRICT,
ADD
  CONSTRAINT `fk_transactions_sender` FOREIGN KEY (`sender`) REFERENCES `wallets` (`wallet_address`) ON DELETE RESTRICT ON UPDATE RESTRICT;