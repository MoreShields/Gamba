-- LoL Summoner Stats
-- Parameters: $1=guild_id, $2=start_date, $3=end_date, $4=min_games_threshold
-- Note: Summoner name is extracted from condition format: "SummonerName - **QueueType**..."

-- Base CTE for LoL games
WITH lol_games AS (
    SELECT
        gw.id,
        gw.condition,
        gw.winning_option_id,
        gw.total_pot,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name,
        wo.option_text as winning_option
    FROM group_wagers gw
    LEFT JOIN group_wager_options wo ON wo.id = gw.winning_option_id
    WHERE gw.guild_id = $1
      AND gw.external_system = 'league_of_legends'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
),
summoner_stats AS (
    SELECT
        summoner_name,
        COUNT(*) as total_games,
        SUM(CASE WHEN winning_option = 'Win' THEN 1 ELSE 0 END) as wins,
        SUM(CASE WHEN winning_option = 'Loss' THEN 1 ELSE 0 END) as losses,
        SUM(total_pot) as total_wagered_on
    FROM lol_games
    WHERE summoner_name IS NOT NULL
    GROUP BY summoner_name
    HAVING COUNT(*) >= $4
)
-- Results for each stat

-- Most Winning Summoner (Win Rate)
SELECT 'lol_best_win_rate' as stat_name,
    summoner_name,
    total_games,
    wins,
    losses,
    ROUND(wins::decimal / total_games * 100, 2) as win_rate_pct,
    total_wagered_on
FROM summoner_stats
ORDER BY (wins::decimal / total_games) DESC
LIMIT 1;

-- Least Winning Summoner (Win Rate)
WITH lol_games AS (
    SELECT
        gw.id,
        gw.condition,
        gw.winning_option_id,
        gw.total_pot,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name,
        wo.option_text as winning_option
    FROM group_wagers gw
    LEFT JOIN group_wager_options wo ON wo.id = gw.winning_option_id
    WHERE gw.guild_id = $1
      AND gw.external_system = 'league_of_legends'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
),
summoner_stats AS (
    SELECT
        summoner_name,
        COUNT(*) as total_games,
        SUM(CASE WHEN winning_option = 'Win' THEN 1 ELSE 0 END) as wins,
        SUM(CASE WHEN winning_option = 'Loss' THEN 1 ELSE 0 END) as losses,
        SUM(total_pot) as total_wagered_on
    FROM lol_games
    WHERE summoner_name IS NOT NULL
    GROUP BY summoner_name
    HAVING COUNT(*) >= $4
)
SELECT 'lol_worst_win_rate' as stat_name,
    summoner_name,
    total_games,
    wins,
    losses,
    ROUND(wins::decimal / total_games * 100, 2) as win_rate_pct,
    total_wagered_on
FROM summoner_stats
ORDER BY (wins::decimal / total_games) ASC
LIMIT 1;

-- Most Profitable Summoner (for bettors)
WITH lol_games AS (
    SELECT
        gw.id as wager_id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
    FROM group_wagers gw
    WHERE gw.guild_id = $1
      AND gw.external_system = 'league_of_legends'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
),
bettor_outcomes AS (
    SELECT
        lg.summoner_name,
        p.discord_id,
        p.amount as bet_amount,
        COALESCE(p.payout_amount, 0) as payout,
        COALESCE(p.payout_amount, 0) - p.amount as net_result
    FROM lol_games lg
    JOIN group_wager_participants p ON p.group_wager_id = lg.wager_id
    WHERE lg.summoner_name IS NOT NULL
)
SELECT 'lol_most_profitable' as stat_name,
    summoner_name,
    COUNT(DISTINCT discord_id) as unique_bettors,
    SUM(bet_amount) as total_wagered,
    SUM(payout) as total_payouts,
    SUM(net_result) as net_bettor_profit
FROM bettor_outcomes
GROUP BY summoner_name
ORDER BY net_bettor_profit DESC
LIMIT 1;

-- Most Costly Summoner (for bettors)
WITH lol_games AS (
    SELECT
        gw.id as wager_id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
    FROM group_wagers gw
    WHERE gw.guild_id = $1
      AND gw.external_system = 'league_of_legends'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
),
bettor_outcomes AS (
    SELECT
        lg.summoner_name,
        p.discord_id,
        p.amount as bet_amount,
        COALESCE(p.payout_amount, 0) as payout,
        COALESCE(p.payout_amount, 0) - p.amount as net_result
    FROM lol_games lg
    JOIN group_wager_participants p ON p.group_wager_id = lg.wager_id
    WHERE lg.summoner_name IS NOT NULL
)
SELECT 'lol_most_costly' as stat_name,
    summoner_name,
    COUNT(DISTINCT discord_id) as unique_bettors,
    SUM(bet_amount) as total_wagered,
    SUM(payout) as total_payouts,
    SUM(net_result) as net_bettor_loss
FROM bettor_outcomes
GROUP BY summoner_name
ORDER BY net_bettor_loss ASC
LIMIT 1;

-- Most Bet-On Summoner
WITH lol_games AS (
    SELECT
        gw.id as wager_id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name,
        gw.total_pot
    FROM group_wagers gw
    WHERE gw.guild_id = $1
      AND gw.external_system = 'league_of_legends'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
)
SELECT 'lol_most_bet_on' as stat_name,
    summoner_name,
    COUNT(*) as total_games,
    SUM(total_pot) as total_wagered
FROM lol_games
WHERE summoner_name IS NOT NULL
GROUP BY summoner_name
ORDER BY total_wagered DESC
LIMIT 1;

-- Most Games Tracked
WITH lol_games AS (
    SELECT
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
    FROM group_wagers gw
    WHERE gw.guild_id = $1
      AND gw.external_system = 'league_of_legends'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
)
SELECT 'lol_most_games_tracked' as stat_name,
    summoner_name,
    COUNT(*) as total_games
FROM lol_games
WHERE summoner_name IS NOT NULL
GROUP BY summoner_name
ORDER BY total_games DESC
LIMIT 1;
