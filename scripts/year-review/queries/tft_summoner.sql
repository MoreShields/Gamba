-- TFT Summoner Stats
-- Parameters: $1=guild_id, $2=start_date, $3=end_date, $4=min_games_threshold
-- Note: Summoner name is extracted from condition format: "SummonerName - **QueueType**..."

-- Best Average Placement
WITH tft_games AS (
    SELECT
        gw.id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name,
        wo.option_text as winning_option,
        CASE wo.option_text
            WHEN '1' THEN 1.0
            WHEN '2' THEN 2.0
            WHEN '3' THEN 3.0
            WHEN '4' THEN 4.0
            WHEN '1-2' THEN 1.5
            WHEN '3-4' THEN 3.5
            WHEN '5-6' THEN 5.5
            WHEN '7-8' THEN 7.5
        END as placement_value
    FROM group_wagers gw
    JOIN group_wager_options wo ON wo.id = gw.winning_option_id
    WHERE gw.guild_id = $1
      AND gw.external_system = 'teamfight_tactics'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
),
summoner_stats AS (
    SELECT
        summoner_name,
        COUNT(*) as total_games,
        AVG(placement_value) as avg_placement,
        SUM(CASE WHEN placement_value <= 4 THEN 1 ELSE 0 END) as top_4_count,
        SUM(CASE WHEN placement_value > 4 THEN 1 ELSE 0 END) as bottom_4_count
    FROM tft_games
    WHERE placement_value IS NOT NULL AND summoner_name IS NOT NULL
    GROUP BY summoner_name
    HAVING COUNT(*) >= $4
)
SELECT 'tft_best_avg_placement' as stat_name,
    summoner_name,
    total_games,
    ROUND(avg_placement::decimal, 2) as avg_placement,
    top_4_count,
    bottom_4_count,
    ROUND(top_4_count::decimal / total_games * 100, 2) as top_4_rate_pct
FROM summoner_stats
ORDER BY avg_placement ASC
LIMIT 1;

-- Worst Average Placement
WITH tft_games AS (
    SELECT
        gw.id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name,
        wo.option_text as winning_option,
        CASE wo.option_text
            WHEN '1' THEN 1.0
            WHEN '2' THEN 2.0
            WHEN '3' THEN 3.0
            WHEN '4' THEN 4.0
            WHEN '1-2' THEN 1.5
            WHEN '3-4' THEN 3.5
            WHEN '5-6' THEN 5.5
            WHEN '7-8' THEN 7.5
        END as placement_value
    FROM group_wagers gw
    JOIN group_wager_options wo ON wo.id = gw.winning_option_id
    WHERE gw.guild_id = $1
      AND gw.external_system = 'teamfight_tactics'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
),
summoner_stats AS (
    SELECT
        summoner_name,
        COUNT(*) as total_games,
        AVG(placement_value) as avg_placement,
        SUM(CASE WHEN placement_value <= 4 THEN 1 ELSE 0 END) as top_4_count,
        SUM(CASE WHEN placement_value > 4 THEN 1 ELSE 0 END) as bottom_4_count
    FROM tft_games
    WHERE placement_value IS NOT NULL AND summoner_name IS NOT NULL
    GROUP BY summoner_name
    HAVING COUNT(*) >= $4
)
SELECT 'tft_worst_avg_placement' as stat_name,
    summoner_name,
    total_games,
    ROUND(avg_placement::decimal, 2) as avg_placement,
    top_4_count,
    bottom_4_count,
    ROUND(top_4_count::decimal / total_games * 100, 2) as top_4_rate_pct
FROM summoner_stats
ORDER BY avg_placement DESC
LIMIT 1;

-- Most Top-4 Finishes
WITH tft_games AS (
    SELECT
        gw.id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name,
        wo.option_text as winning_option,
        CASE wo.option_text
            WHEN '1' THEN 1.0
            WHEN '2' THEN 2.0
            WHEN '3' THEN 3.0
            WHEN '4' THEN 4.0
            WHEN '1-2' THEN 1.5
            WHEN '3-4' THEN 3.5
            WHEN '5-6' THEN 5.5
            WHEN '7-8' THEN 7.5
        END as placement_value
    FROM group_wagers gw
    JOIN group_wager_options wo ON wo.id = gw.winning_option_id
    WHERE gw.guild_id = $1
      AND gw.external_system = 'teamfight_tactics'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
)
SELECT 'tft_most_top4' as stat_name,
    summoner_name,
    COUNT(*) as total_games,
    SUM(CASE WHEN placement_value <= 4 THEN 1 ELSE 0 END) as top_4_count
FROM tft_games
WHERE placement_value IS NOT NULL AND summoner_name IS NOT NULL
GROUP BY summoner_name
ORDER BY top_4_count DESC
LIMIT 1;

-- Most Bottom-4 Finishes
WITH tft_games AS (
    SELECT
        gw.id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name,
        wo.option_text as winning_option,
        CASE wo.option_text
            WHEN '1' THEN 1.0
            WHEN '2' THEN 2.0
            WHEN '3' THEN 3.0
            WHEN '4' THEN 4.0
            WHEN '1-2' THEN 1.5
            WHEN '3-4' THEN 3.5
            WHEN '5-6' THEN 5.5
            WHEN '7-8' THEN 7.5
        END as placement_value
    FROM group_wagers gw
    JOIN group_wager_options wo ON wo.id = gw.winning_option_id
    WHERE gw.guild_id = $1
      AND gw.external_system = 'teamfight_tactics'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
)
SELECT 'tft_most_bottom4' as stat_name,
    summoner_name,
    COUNT(*) as total_games,
    SUM(CASE WHEN placement_value > 4 THEN 1 ELSE 0 END) as bottom_4_count
FROM tft_games
WHERE placement_value IS NOT NULL AND summoner_name IS NOT NULL
GROUP BY summoner_name
ORDER BY bottom_4_count DESC
LIMIT 1;

-- Most TFT Games Tracked
WITH tft_games AS (
    SELECT
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
    FROM group_wagers gw
    WHERE gw.guild_id = $1
      AND gw.external_system = 'teamfight_tactics'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
)
SELECT 'tft_most_games_tracked' as stat_name,
    summoner_name,
    COUNT(*) as total_games
FROM tft_games
WHERE summoner_name IS NOT NULL
GROUP BY summoner_name
ORDER BY total_games DESC
LIMIT 1;

-- Most Profitable TFT Summoner (for bettors)
WITH tft_games AS (
    SELECT
        gw.id as wager_id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
    FROM group_wagers gw
    WHERE gw.guild_id = $1
      AND gw.external_system = 'teamfight_tactics'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
),
bettor_outcomes AS (
    SELECT
        tg.summoner_name,
        p.discord_id,
        p.amount as bet_amount,
        COALESCE(p.payout_amount, 0) as payout,
        COALESCE(p.payout_amount, 0) - p.amount as net_result
    FROM tft_games tg
    JOIN group_wager_participants p ON p.group_wager_id = tg.wager_id
    WHERE tg.summoner_name IS NOT NULL
)
SELECT 'tft_most_profitable' as stat_name,
    summoner_name,
    COUNT(DISTINCT discord_id) as unique_bettors,
    SUM(bet_amount) as total_wagered,
    SUM(payout) as total_payouts,
    SUM(net_result) as net_bettor_profit
FROM bettor_outcomes
GROUP BY summoner_name
ORDER BY net_bettor_profit DESC
LIMIT 1;

-- Most Costly TFT Summoner (for bettors)
WITH tft_games AS (
    SELECT
        gw.id as wager_id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
    FROM group_wagers gw
    WHERE gw.guild_id = $1
      AND gw.external_system = 'teamfight_tactics'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
),
bettor_outcomes AS (
    SELECT
        tg.summoner_name,
        p.discord_id,
        p.amount as bet_amount,
        COALESCE(p.payout_amount, 0) as payout,
        COALESCE(p.payout_amount, 0) - p.amount as net_result
    FROM tft_games tg
    JOIN group_wager_participants p ON p.group_wager_id = tg.wager_id
    WHERE tg.summoner_name IS NOT NULL
)
SELECT 'tft_most_costly' as stat_name,
    summoner_name,
    COUNT(DISTINCT discord_id) as unique_bettors,
    SUM(bet_amount) as total_wagered,
    SUM(payout) as total_payouts,
    SUM(net_result) as net_bettor_loss
FROM bettor_outcomes
GROUP BY summoner_name
ORDER BY net_bettor_loss ASC
LIMIT 1;

-- Most Bet-On TFT Summoner
WITH tft_games AS (
    SELECT
        gw.id as wager_id,
        TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name,
        gw.total_pot
    FROM group_wagers gw
    WHERE gw.guild_id = $1
      AND gw.external_system = 'teamfight_tactics'
      AND gw.state = 'resolved'
      AND gw.created_at >= $2
      AND gw.created_at < $3
)
SELECT 'tft_most_bet_on' as stat_name,
    summoner_name,
    COUNT(*) as total_games,
    SUM(total_pot) as total_wagered
FROM tft_games
WHERE summoner_name IS NOT NULL
GROUP BY summoner_name
ORDER BY total_wagered DESC
LIMIT 1;
