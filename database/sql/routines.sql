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


DROP PROCEDURE IF EXISTS `getWalletBalance`;

CREATE DEFINER=`root`@`localhost` PROCEDURE `getWalletBalance`(
  IN `p_wallet_address` VARCHAR(255), 
  OUT `p_account_balance` DECIMAL(10,2)
)   
BEGIN
	DECLARE var_initial_deposit DECIMAL(10,2);
  DECLARE var_total_sent DECIMAL(10,2);
  DECLARE var_total_received DECIMAL(10,2);

  -- Check if wallet exists
  IF NOT EXISTS(SELECT 1 FROM wallets WHERE wallet_address = p_wallet_address) THEN
    SIGNAL SQLSTATE "45000"
    SET MESSAGE_TEXT="Wallet does not exist";
  END IF;

  -- Get initial deposit
  SELECT initial_deposit
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

  SET p_account_balance = var_initial_deposit + var_total_received - var_total_sent;

END;

DROP PROCEDURE IF EXISTS `getTransactionFee`;

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

