

ALTER TABLE `limits`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `wallet_address` (`wallet_address`),
  ADD KEY `fk_limits_user_id` (`user_id`);

ALTER TABLE `password_reset_tokens`
  ADD PRIMARY KEY (`id`);

ALTER TABLE `public_keys`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `public_key_hash` (`public_key_hash`),
  ADD KEY `fk_public_keys_email` (`email`);

ALTER TABLE `request_funds`
  ADD PRIMARY KEY (`id`),
  ADD KEY `sender` (`sender`),
  ADD KEY `receiver` (`receiver`);

ALTER TABLE `transactions`
  ADD PRIMARY KEY (`id`),
  ADD KEY `fk_transactions_receiver` (`receiver`),
  ADD KEY `fk_transactions_sender` (`sender`);

ALTER TABLE `users`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `email` (`email`),
  ADD UNIQUE KEY `phone_no` (`phone_no`);

ALTER TABLE `wallets`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `wallet_address` (`wallet_address`),
  ADD KEY `user_id` (`user_id`);


ALTER TABLE `limits`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

ALTER TABLE `password_reset_tokens`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

ALTER TABLE `public_keys`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

ALTER TABLE `request_funds`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

ALTER TABLE `transactions`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

ALTER TABLE `users`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;

ALTER TABLE `wallets`
  MODIFY `id` bigint NOT NULL AUTO_INCREMENT;


ALTER TABLE `limits`
  ADD CONSTRAINT `fk_limits_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `fk_limits_wallet_address` FOREIGN KEY (`wallet_address`) REFERENCES `wallets` (`wallet_address`) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE `public_keys`
  ADD CONSTRAINT `fk_public_keys_email` FOREIGN KEY (`email`) REFERENCES `users` (`email`) ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE `request_funds`
  ADD CONSTRAINT `fk_request_funds_receiver` FOREIGN KEY (`receiver`) REFERENCES `wallets` (`wallet_address`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `fk_request_funds_sender` FOREIGN KEY (`sender`) REFERENCES `wallets` (`wallet_address`) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE `transactions`
  ADD CONSTRAINT `fk_transactions_receiver` FOREIGN KEY (`receiver`) REFERENCES `wallets` (`wallet_address`) ON DELETE RESTRICT ON UPDATE RESTRICT,
  ADD CONSTRAINT `fk_transactions_sender` FOREIGN KEY (`sender`) REFERENCES `wallets` (`wallet_address`) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE `wallets`
  ADD CONSTRAINT `wallets_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`);
