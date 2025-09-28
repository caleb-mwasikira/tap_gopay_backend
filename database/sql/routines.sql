DROP PROCEDURE IF EXISTS `getTotalAmountSpent`;

CREATE DEFINER=`root`@`localhost` PROCEDURE `getTotalAmountSpent`(
  IN `p_wallet_address` VARCHAR(32)
)
BEGIN
    -- Last 7 days
    SELECT
        'week' AS period,
        COALESCE(SUM(amount), 0) AS total_amount
    FROM transactions
    WHERE sender = p_wallet_address
      AND created_at >= NOW() - INTERVAL 7 DAY

    UNION ALL

    -- Last 1 month
    SELECT
        'month' AS period,
        COALESCE(SUM(amount), 0) AS total_amount
    FROM transactions
    WHERE sender = p_wallet_address
      AND created_at >= NOW() - INTERVAL 1 MONTH

    UNION ALL

    -- Last 1 year
    SELECT
        'year' AS period,
        COALESCE(SUM(amount), 0) AS total_amount
    FROM transactions
    WHERE sender = p_wallet_address
      AND created_at >= NOW() - INTERVAL 1 YEAR;
END;

CREATE DEFINER=`root`@`localhost` PROCEDURE `getTransactionFee`(
  IN `p_amount` DECIMAL(10,2),
  OUT `p_fee` DECIMAL(10,2)
)
BEGIN
  SELECT COALESCE(fee, 0)
  INTO p_fee
  FROM transaction_fees
  WHERE p_amount BETWEEN min_amount AND max_amount
  AND NOW() BETWEEN effective_from AND COALESCE(effective_to, NOW())
  LIMIT 1;
END;

CREATE DEFINER=`root`@`localhost` PROCEDURE `getWalletBalance`(IN `p_wallet_address` VARCHAR(255), OUT `p_wallet_balance` DECIMAL(10,2))
BEGIN
  DECLARE var_initial_deposit DECIMAL(10,2) DEFAULT 0;
  DECLARE var_total_sent DECIMAL(10,2);
  DECLARE var_total_received DECIMAL(10,2);

  -- Check if wallet exists
  CALL walletExists(p_wallet_address, @wallet_exists, @wallet_active);

  IF NOT @wallet_exists THEN
    SIGNAL SQLSTATE "45000"
    SET MESSAGE_TEXT="Wallet does NOT exist";
  END IF;

  -- Get initial deposit
  -- Remember to COALESCE NULL values to 0 as the wallet address
  -- may not be found in wallets table (as is the case with cash pool wallets)
  SELECT COALESCE(initial_deposit, 0)
  INTO var_initial_deposit
  FROM wallets
  WHERE wallet_address = p_wallet_address
  LIMIT 1;

  -- Get total amount sent
  SELECT COALESCE(SUM(amount), 0)
  INTO var_total_sent
  FROM transactions
  WHERE sender = p_wallet_address;

  -- Get total amount received
  SELECT COALESCE(SUM(amount), 0)
  INTO var_total_received
  FROM transactions
  WHERE receiver = p_wallet_address;

  SET p_wallet_balance = var_initial_deposit + var_total_received - var_total_sent;

END;

CREATE DEFINER=`root`@`localhost` PROCEDURE `walletExists`(
  IN `p_wallet_address` VARCHAR(255),
  OUT `p_wallet_exists` BOOLEAN,
  OUT `p_wallet_active` BOOLEAN
)
BEGIN
    DECLARE wallet_exists TINYINT DEFAULT 0;
    DECLARE wallet_active TINYINT DEFAULT 0;

    -- First check if wallet exists in wallets
    SELECT COUNT(*) > 0 INTO wallet_exists
    FROM wallets
    WHERE wallet_address = p_wallet_address;

    -- If exists, check if it's active
    IF wallet_exists = 1 THEN
        SELECT is_active INTO wallet_active
        FROM wallets
        WHERE wallet_address = p_wallet_address
        LIMIT 1;
    ELSE
        -- Otherwise check cash_pools
        SELECT COUNT(*) > 0 INTO wallet_exists
        FROM cash_pools
        WHERE wallet_address = p_wallet_address;

        IF wallet_exists = 1 THEN
            SELECT expires_at > NOW() INTO wallet_active
            FROM cash_pools
            WHERE wallet_address = p_wallet_address
            LIMIT 1;
        END IF;
    END IF;

    SET p_wallet_exists = wallet_exists;
    SET p_wallet_active = wallet_active;
END;