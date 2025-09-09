CREATE VIEW transaction_details AS SELECT
    t.transaction_id,
    us.username AS senders_username,
    us.phone_no AS senders_phone,
    cs.card_no AS senders_card_no,
    ur.username AS receivers_username,
    ur.phone_no AS receivers_phone,
    cr.card_no AS receivers_card_no,
    t.amount,
    t.signature,
    t.created_at
FROM
    transactions t
JOIN credit_cards cs ON
    t.sender = cs.card_no
JOIN users us ON
    cs.user_id = us.id
JOIN credit_cards cr ON
    t.receiver = cr.card_no
JOIN users ur ON
    cr.user_id = ur.id;