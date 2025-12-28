-- Win/Loss Records Stats
-- Parameters: $1=guild_id, $2=start_date, $3=end_date, $4=min_bet_threshold, $5=min_wager_threshold

-- Most Bets Placed
SELECT 'most_bets_placed' as stat_name,
    discord_id,
    COUNT(*) as bet_count,
    SUM(amount) as total_wagered,
    SUM(CASE WHEN won THEN 1 ELSE 0 END) as wins,
    ROUND(AVG(CASE WHEN won THEN 1.0 ELSE 0.0 END) * 100, 2) as win_rate_pct
FROM bets
WHERE guild_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY discord_id
ORDER BY bet_count DESC
LIMIT 1;

-- Most Wagers Proposed
SELECT 'most_wagers_proposed' as stat_name,
    proposer_discord_id as discord_id,
    COUNT(*) as wager_count
FROM wagers
WHERE guild_id = $1
  AND created_at >= $2
  AND created_at < $3
GROUP BY proposer_discord_id
ORDER BY wager_count DESC
LIMIT 1;

-- Best Bet Win Rate (min threshold)
WITH user_stats AS (
    SELECT
        discord_id,
        COUNT(*) as total_bets,
        SUM(CASE WHEN won THEN 1 ELSE 0 END) as wins
    FROM bets
    WHERE guild_id = $1
      AND created_at >= $2
      AND created_at < $3
    GROUP BY discord_id
    HAVING COUNT(*) >= $4
)
SELECT 'best_bet_win_rate' as stat_name,
    discord_id,
    total_bets,
    wins,
    ROUND(wins::decimal / total_bets * 100, 2) as win_rate_pct
FROM user_stats
ORDER BY (wins::decimal / total_bets) DESC
LIMIT 1;

-- Worst Bet Win Rate (min threshold)
WITH user_stats AS (
    SELECT
        discord_id,
        COUNT(*) as total_bets,
        SUM(CASE WHEN won THEN 1 ELSE 0 END) as wins
    FROM bets
    WHERE guild_id = $1
      AND created_at >= $2
      AND created_at < $3
    GROUP BY discord_id
    HAVING COUNT(*) >= $4
)
SELECT 'worst_bet_win_rate' as stat_name,
    discord_id,
    total_bets,
    wins,
    ROUND(wins::decimal / total_bets * 100, 2) as win_rate_pct
FROM user_stats
ORDER BY (wins::decimal / total_bets) ASC
LIMIT 1;

-- Most H2H Wager Wins
SELECT 'most_h2h_wager_wins' as stat_name,
    winner_discord_id as discord_id,
    COUNT(*) as win_count
FROM wagers
WHERE guild_id = $1
  AND state = 'resolved'
  AND winner_discord_id IS NOT NULL
  AND created_at >= $2
  AND created_at < $3
GROUP BY winner_discord_id
ORDER BY win_count DESC
LIMIT 1;

-- Most H2H Wager Losses
WITH participants AS (
    SELECT id, proposer_discord_id as discord_id FROM wagers
    WHERE guild_id = $1 AND state = 'resolved' AND created_at >= $2 AND created_at < $3
    UNION ALL
    SELECT id, target_discord_id as discord_id FROM wagers
    WHERE guild_id = $1 AND state = 'resolved' AND created_at >= $2 AND created_at < $3
),
losses AS (
    SELECT p.discord_id, COUNT(*) as loss_count
    FROM participants p
    JOIN wagers w ON w.id = p.id
    WHERE w.winner_discord_id IS NOT NULL
      AND w.winner_discord_id != p.discord_id
    GROUP BY p.discord_id
)
SELECT 'most_h2h_wager_losses' as stat_name,
    discord_id,
    loss_count
FROM losses
ORDER BY loss_count DESC
LIMIT 1;

-- Most Group Wager Wins
SELECT 'most_group_wager_wins' as stat_name,
    p.discord_id,
    COUNT(*) as win_count,
    SUM(p.payout_amount - p.amount) as total_profit
FROM group_wager_participants p
JOIN group_wagers gw ON gw.id = p.group_wager_id
WHERE gw.guild_id = $1
  AND gw.state = 'resolved'
  AND gw.created_at >= $2
  AND gw.created_at < $3
  AND p.payout_amount > p.amount
GROUP BY p.discord_id
ORDER BY win_count DESC
LIMIT 1;
