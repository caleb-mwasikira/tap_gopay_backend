--
-- Database: `tap_gopay`
--
-- --------------------------------------------------------
--
-- Table structure for table `signatures`
--
CREATE TABLE
  `signatures` (
    `id` bigint NOT NULL,
    `transaction_code` bigint NOT NULL,
    `user_id` bigint NOT NULL,
    `signature` text NOT NULL,
    `public_key_hash` varchar(255) NOT NULL
  ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TRIGGER `updateSignatureCount` AFTER INSERT ON `signatures` FOR EACH ROW BEGIN
UPDATE transactions
SET
  signatures_count = signatures_count + 1
WHERE
  transaction_code = NEW.transaction_code;

END;

CREATE TRIGGER `verifySignature` BEFORE INSERT ON `signatures` FOR EACH ROW BEGIN DECLARE var_senders_wallet_address VARCHAR(255);

-- Get sender’s wallet address
SELECT
  sender INTO var_senders_wallet_address
FROM
  transactions
WHERE
  transaction_code = NEW.transaction_code;

-- Check ownership
IF NOT EXISTS (
  SELECT
    1
  FROM
    wallet_owners
  WHERE
    wallet_address = var_senders_wallet_address
    AND user_id = NEW.user_id
) THEN SIGNAL SQLSTATE '45000'
SET
  MESSAGE_TEXT = 'This wallet does not belong to you';

END IF;

END;