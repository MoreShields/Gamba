-- Platform Totals Stats
-- Parameters: $1=guild_id, $2=start_date, $3=end_date

-- Total Bets Placed
SELECT 'total_bets_placed' as stat_name,
    COUNT(*) as total_count
FROM bets
WHERE guild_id = $1
  AND created_at >= $2
  AND created_at < $3;

-- Total Amount Wagered (bets)
SELECT 'total_amount_wagered' as stat_name,
    COALESCE(SUM(amount), 0) as total_amount
FROM bets
WHERE guild_id = $1
  AND created_at >= $2
  AND created_at < $3;

-- Total Wagers Created (H2H)
SELECT 'total_wagers_created' as stat_name,
    COUNT(*) as total_count
FROM wagers
WHERE guild_id = $1
  AND created_at >= $2
  AND created_at < $3;

-- Total Group Wagers
SELECT 'total_group_wagers' as stat_name,
    COUNT(*) as total_count
FROM group_wagers
WHERE guild_id = $1
  AND created_at >= $2
  AND created_at < $3;
