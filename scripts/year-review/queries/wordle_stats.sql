-- Wordle Stats
-- Parameters: $1=guild_id, $2=start_date, $3=end_date, $4=min_completions_threshold

-- Longest Wordle Streak
WITH daily_completions AS (
    SELECT
        discord_id,
        DATE(completed_at) as completion_date,
        ROW_NUMBER() OVER (PARTITION BY discord_id ORDER BY DATE(completed_at)) as row_num
    FROM wordle_completions
    WHERE guild_id = $1
      AND completed_at >= $2
      AND completed_at < $3
),
streak_groups AS (
    SELECT
        discord_id,
        completion_date,
        completion_date - (row_num * INTERVAL '1 day')::date as streak_group
    FROM daily_completions
),
streaks AS (
    SELECT
        discord_id,
        COUNT(*) as streak_length,
        MIN(completion_date) as streak_start,
        MAX(completion_date) as streak_end
    FROM streak_groups
    GROUP BY discord_id, streak_group
)
SELECT 'longest_wordle_streak' as stat_name,
    discord_id,
    streak_length,
    streak_start,
    streak_end
FROM streaks
ORDER BY streak_length DESC
LIMIT 1;

-- Most Wordle Completions
SELECT 'most_wordle_completions' as stat_name,
    discord_id,
    COUNT(*) as completion_count
FROM wordle_completions
WHERE guild_id = $1
  AND completed_at >= $2
  AND completed_at < $3
GROUP BY discord_id
ORDER BY completion_count DESC
LIMIT 1;

-- Best Average Guess Count (min completions threshold)
SELECT 'best_avg_guess_count' as stat_name,
    discord_id,
    COUNT(*) as completion_count,
    ROUND(AVG(guess_count)::decimal, 2) as avg_guesses
FROM wordle_completions
WHERE guild_id = $1
  AND completed_at >= $2
  AND completed_at < $3
GROUP BY discord_id
HAVING COUNT(*) >= $4
ORDER BY AVG(guess_count) ASC
LIMIT 1;

-- Most Wordle Rewards Earned
SELECT 'most_wordle_rewards' as stat_name,
    discord_id,
    COUNT(*) as reward_count,
    SUM(change_amount) as total_rewards
FROM balance_history
WHERE guild_id = $1
  AND transaction_type = 'wordle_reward'
  AND created_at >= $2
  AND created_at < $3
GROUP BY discord_id
ORDER BY total_rewards DESC
LIMIT 1;
