-- High Roller Stats
-- Parameters: $1=guild_id, $2=start_date, $3=end_date

-- Most High Roller Purchases
SELECT 'most_high_roller_purchases' as stat_name,
    discord_id,
    COUNT(*) as purchase_count,
    SUM(purchase_price) as total_spent
FROM high_roller_purchases
WHERE guild_id = $1
  AND purchased_at >= $2
  AND purchased_at < $3
GROUP BY discord_id
ORDER BY purchase_count DESC
LIMIT 1;

-- Highest Total Spent on High Roller
SELECT 'highest_high_roller_spent' as stat_name,
    discord_id,
    COUNT(*) as purchase_count,
    SUM(purchase_price) as total_spent
FROM high_roller_purchases
WHERE guild_id = $1
  AND purchased_at >= $2
  AND purchased_at < $3
GROUP BY discord_id
ORDER BY total_spent DESC
LIMIT 1;
