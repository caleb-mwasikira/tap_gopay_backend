DELIMITER $$

-- Gets a summary of all transactions carried out on a credit card
-- for the past 7 days, 1 month and 1 year.
-- This procedure is going to be used to set limits on the amount
-- of spending allowed on a credit card.
CREATE PROCEDURE getTotalAmountSpent(IN p_card_no VARCHAR(32))
BEGIN
    -- Last 7 days
    SELECT
        'week' AS period,
        COALESCE(SUM(amount), 0) AS total_amount
    FROM transactions
    WHERE sender = p_card_no
      AND created_at >= NOW() - INTERVAL 7 DAY

    UNION ALL

    -- Last 1 month
    SELECT
        'month' AS period,
        COALESCE(SUM(amount), 0) AS total_amount
    FROM transactions
    WHERE sender = p_card_no
      AND created_at >= NOW() - INTERVAL 1 MONTH

    UNION ALL

    -- Last 1 year
    SELECT
        'year' AS period,
        COALESCE(SUM(amount), 0) AS total_amount
    FROM transactions
    WHERE sender = p_card_no
      AND created_at >= NOW() - INTERVAL 1 YEAR;
END$$

DELIMITER ;
