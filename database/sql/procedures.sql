--
-- getTotalAmountSpent
--

CREATE PROCEDURE `getTotalAmountSpent`(IN `p_wallet_address` VARCHAR(32))
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


--
-- getWalletBalance
--

CREATE PROCEDURE `getWalletBalance`(IN `p_wallet_address` VARCHAR(255), OUT `account_balance` DECIMAL(10,2))
BEGIN
	DECLARE var_initial_deposit DECIMAL(10,2);
    DECLARE amount_sent DECIMAL(10,2);
    DECLARE amount_received DECIMAL(10,2);

	-- Check if wallet exists
    IF NOT EXISTS(SELECT 1 FROM wallets WHERE wallet_address = p_wallet_address) THEN
        SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = "Wallet does not exist";
    END IF;

    -- Get total amount sent by wallet
    SELECT
        COALESCE(SUM(amount), 0)
    INTO amount_sent
    FROM transactions
    WHERE sender = p_wallet_address;

	-- Get total amount received by wallet
    SELECT
        COALESCE(SUM(amount), 0)
    INTO amount_received
    FROM transactions
    WHERE receiver = p_wallet_address;

    -- Get wallets initial deposit
    SELECT initial_deposit
    INTO var_initial_deposit
    FROM wallets
    WHERE wallet_address = p_wallet_address;

    -- Calculate the current balance
    SET account_balance = var_initial_deposit + amount_received - amount_sent;
END;
