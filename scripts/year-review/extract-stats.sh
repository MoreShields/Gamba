#!/bin/bash
# Year in Review Stats Extraction Script
# Usage: ./extract-stats.sh <guild_id> [year]
# Requires: psql, jq
# Env: DATABASE_URL

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
QUERIES_DIR="${SCRIPT_DIR}/queries"
DATABASE_URL="postgresql:///gamba_db?host=/var/run/postgresql"

# Parse arguments
GUILD_ID="${1:-}"
YEAR="${2:-$(date +%Y)}"

if [ -z "$GUILD_ID" ]; then
    echo "Usage: $0 <guild_id> [year]" >&2
    echo "  guild_id: Discord guild ID (required)" >&2
    echo "  year: Year to generate stats for (default: current year)" >&2
    exit 1
fi

if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL environment variable is not set" >&2
    exit 1
fi

# Date range for the year
START_DATE="${YEAR}-01-01"
END_DATE="$((YEAR + 1))-01-01"

# Thresholds
MIN_BET_THRESHOLD=10
MIN_WAGER_THRESHOLD=3
MIN_WORDLE_THRESHOLD=10
MIN_GAMES_THRESHOLD=5

# Function to run a query and return JSON
run_query() {
    local sql="$1"
    psql "$DATABASE_URL" -t -A -F',' -c "$sql" 2>/dev/null || echo ""
}

# Function to run a SQL file and capture all results as JSON array
run_sql_file() {
    local file="$1"
    local results="[]"

    # Read and execute each statement separated by semicolons
    # Replace parameters with actual values
    local sql_content
    sql_content=$(cat "$file" | \
        sed "s/\\\$1/'$GUILD_ID'/g" | \
        sed "s/\\\$2/'$START_DATE'/g" | \
        sed "s/\\\$3/'$END_DATE'/g" | \
        sed "s/\\\$4/$MIN_BET_THRESHOLD/g" | \
        sed "s/\\\$5/$MIN_WAGER_THRESHOLD/g")

    # Execute and get JSON output
    psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            $sql_content
        ) t;
    " 2>/dev/null | tr -d '[:space:]' || echo "null"
}

# Function to run individual stat queries and build JSON
run_stat_query() {
    local stat_name="$1"
    local sql="$2"

    # Replace parameters
    sql=$(echo "$sql" | \
        sed "s/\\\$1/'$GUILD_ID'/g" | \
        sed "s/\\\$2/'$START_DATE'/g" | \
        sed "s/\\\$3/'$END_DATE'/g" | \
        sed "s/\\\$4/$MIN_BET_THRESHOLD/g" | \
        sed "s/\\\$5/$MIN_WAGER_THRESHOLD/g")

    local result
    result=$(psql "$DATABASE_URL" -t -c "
        SELECT row_to_json(t)
        FROM ($sql) t;
    " 2>/dev/null | tr -d '\n' | sed 's/^[[:space:]]*//')

    # Return null if empty or whitespace only
    if [ -z "$result" ] || [ "$result" = "" ]; then
        echo "null"
    else
        echo "$result"
    fi
}

# Collect all stats into JSON structure
collect_stats() {
    local json_output=""

    # Start JSON object
    json_output="{"
    json_output+="\"guild_id\": \"$GUILD_ID\","
    json_output+="\"year\": $YEAR,"
    json_output+="\"generated_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\","
    json_output+="\"date_range\": {\"start\": \"$START_DATE\", \"end\": \"$END_DATE\"},"
    json_output+="\"thresholds\": {\"min_bets\": $MIN_BET_THRESHOLD, \"min_wagers\": $MIN_WAGER_THRESHOLD, \"min_wordle\": $MIN_WORDLE_THRESHOLD, \"min_games\": $MIN_GAMES_THRESHOLD},"

    # Total Server Activity (sum of absolute balance changes)
    json_output+="\"total_server_activity\": $(run_stat_query "total_server_activity" "
        SELECT COALESCE(SUM(ABS(change_amount)), 0) as total_volume
        FROM balance_history
        WHERE guild_id = \$1 AND created_at >= \$2 AND created_at < \$3
    "),"

    # Gambling Stats
    json_output+="\"gambling\": {"

    # Top 3 Most Bits Wagered - sorted by total wagered
    local top_bits_wagered
    top_bits_wagered=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            SELECT discord_id, COUNT(*) as bet_count, SUM(amount) as total_wagered
            FROM bets WHERE guild_id = '$GUILD_ID' AND created_at >= '$START_DATE' AND created_at < '$END_DATE'
            GROUP BY discord_id ORDER BY SUM(amount) DESC LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"top_bits_wagered\": ${top_bits_wagered:-null},"

    json_output+="\"most_h2h_wager_wins\": $(run_stat_query "most_h2h_wager_wins" "
        SELECT winner_discord_id as discord_id, COUNT(*) as win_count
        FROM wagers WHERE guild_id = \$1 AND state = 'resolved' AND winner_discord_id IS NOT NULL
            AND created_at >= \$2 AND created_at < \$3
        GROUP BY winner_discord_id ORDER BY win_count DESC LIMIT 1
    "),"

    json_output+="\"total_bets_placed\": $(run_stat_query "total_bets_placed" "
        SELECT COUNT(*) as count FROM bets
        WHERE guild_id = \$1 AND created_at >= \$2 AND created_at < \$3
    "),"

    json_output+="\"total_amount_wagered\": $(run_stat_query "total_amount_wagered" "
        SELECT COALESCE(SUM(amount), 0) as amount FROM bets
        WHERE guild_id = \$1 AND created_at >= \$2 AND created_at < \$3
    "),"

    # Top 3 Most 10% Odds Hits
    local top_10_percent_hits
    top_10_percent_hits=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            SELECT discord_id, COUNT(*) as hit_count
            FROM bets
            WHERE guild_id = '$GUILD_ID' AND created_at >= '$START_DATE' AND created_at < '$END_DATE'
                AND win_probability = 0.1 AND won = true
            GROUP BY discord_id ORDER BY hit_count DESC LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"top_10_percent_hits\": ${top_10_percent_hits:-null},"

    # Top 3 Biggest Winners (by net profit from gambling)
    local top_winners
    top_winners=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            SELECT discord_id, COUNT(*) as bet_count,
                SUM(amount) as total_wagered,
                SUM(CASE WHEN won = true THEN amount * (1.0/win_probability - 1.0) ELSE -amount END)::bigint as net_profit
            FROM bets
            WHERE guild_id = '$GUILD_ID' AND created_at >= '$START_DATE' AND created_at < '$END_DATE'
            GROUP BY discord_id
            HAVING COUNT(*) >= $MIN_BET_THRESHOLD
            ORDER BY SUM(CASE WHEN won = true THEN amount * (1.0/win_probability - 1.0) ELSE -amount END) DESC
            LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"top_winners\": ${top_winners:-null},"

    # Top 3 Biggest Losers (by net profit from gambling)
    local top_losers
    top_losers=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            SELECT discord_id, COUNT(*) as bet_count,
                SUM(amount) as total_wagered,
                SUM(CASE WHEN won = true THEN amount * (1.0/win_probability - 1.0) ELSE -amount END)::bigint as net_profit
            FROM bets
            WHERE guild_id = '$GUILD_ID' AND created_at >= '$START_DATE' AND created_at < '$END_DATE'
            GROUP BY discord_id
            HAVING COUNT(*) >= $MIN_BET_THRESHOLD
            ORDER BY SUM(CASE WHEN won = true THEN amount * (1.0/win_probability - 1.0) ELSE -amount END) ASC
            LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"top_losers\": ${top_losers:-null}"

    json_output+="},"

    # Group Wagers (User-Submitted Only - excludes LoL/TFT)
    json_output+="\"group_wagers\": {"

    json_output+="\"most_created\": $(run_stat_query "most_created" "
        SELECT creator_discord_id as discord_id, COUNT(*) as created_count
        FROM group_wagers
        WHERE guild_id = \$1 AND external_system IS NULL
            AND created_at >= \$2 AND created_at < \$3
        GROUP BY creator_discord_id ORDER BY created_count DESC LIMIT 1
    "),"

    json_output+="\"most_wins\": $(run_stat_query "most_wins" "
        SELECT p.discord_id, COUNT(*) as win_count, SUM(p.payout_amount - p.amount) as total_profit
        FROM group_wager_participants p
        JOIN group_wagers gw ON gw.id = p.group_wager_id
        WHERE gw.guild_id = \$1 AND gw.external_system IS NULL AND gw.state = 'resolved'
            AND gw.created_at >= \$2 AND gw.created_at < \$3
            AND p.payout_amount > p.amount
        GROUP BY p.discord_id ORDER BY win_count DESC LIMIT 1
    "),"

    json_output+="\"most_participation\": $(run_stat_query "most_participation" "
        SELECT p.discord_id, COUNT(*) as participation_count
        FROM group_wager_participants p
        JOIN group_wagers gw ON gw.id = p.group_wager_id
        WHERE gw.guild_id = \$1 AND gw.external_system IS NULL
            AND gw.created_at >= \$2 AND gw.created_at < \$3
        GROUP BY p.discord_id ORDER BY participation_count DESC LIMIT 1
    "),"

    json_output+="\"total_group_wagers\": $(run_stat_query "total_group_wagers" "
        SELECT COUNT(*) as count FROM group_wagers
        WHERE guild_id = \$1 AND external_system IS NULL
            AND created_at >= \$2 AND created_at < \$3
    "),"

    json_output+="\"total_amount_wagered\": $(run_stat_query "total_amount_wagered" "
        SELECT COALESCE(SUM(p.amount), 0) as amount
        FROM group_wager_participants p
        JOIN group_wagers gw ON gw.id = p.group_wager_id
        WHERE gw.guild_id = \$1 AND gw.external_system IS NULL
            AND gw.created_at >= \$2 AND gw.created_at < \$3
    ")"

    json_output+="},"

    # Wordle Stats
    json_output+="\"wordle_stats\": {"

    json_output+="\"longest_streak\": $(run_stat_query "longest_streak" "
        WITH daily AS (
            SELECT discord_id, DATE(completed_at) as d,
                ROW_NUMBER() OVER (PARTITION BY discord_id ORDER BY DATE(completed_at)) as rn
            FROM wordle_completions
            WHERE guild_id = \$1 AND completed_at >= \$2 AND completed_at < \$3
        ),
        groups AS (
            SELECT discord_id, d, d - rn::int as grp
            FROM daily
        ),
        streaks AS (
            SELECT discord_id, COUNT(*) as streak_length, MIN(d) as streak_start, MAX(d) as streak_end
            FROM groups GROUP BY discord_id, grp
        )
        SELECT discord_id, streak_length, streak_start, streak_end
        FROM streaks ORDER BY streak_length DESC LIMIT 1
    "),"

    json_output+="\"most_completions\": $(run_stat_query "most_completions" "
        SELECT discord_id, COUNT(*) as completion_count
        FROM wordle_completions
        WHERE guild_id = \$1 AND completed_at >= \$2 AND completed_at < \$3
        GROUP BY discord_id ORDER BY completion_count DESC LIMIT 1
    "),"

    json_output+="\"best_avg_guesses\": $(run_stat_query "best_avg_guesses" "
        SELECT discord_id, COUNT(*) as completion_count, ROUND(AVG(guess_count)::decimal, 2) as avg_guesses
        FROM wordle_completions
        WHERE guild_id = \$1 AND completed_at >= \$2 AND completed_at < \$3
        GROUP BY discord_id HAVING COUNT(*) >= $MIN_WORDLE_THRESHOLD
        ORDER BY AVG(guess_count) ASC LIMIT 1
    "),"

    json_output+="\"most_rewards\": $(run_stat_query "most_rewards" "
        SELECT discord_id, COUNT(*) as reward_count, SUM(change_amount) as total_rewards
        FROM balance_history
        WHERE guild_id = \$1 AND transaction_type = 'wordle_reward'
            AND created_at >= \$2 AND created_at < \$3
        GROUP BY discord_id ORDER BY total_rewards DESC LIMIT 1
    ")"

    json_output+="},"

    # High Roller Stats
    json_output+="\"high_roller\": {"

    # Top 3 Highest Total Spent on High Roller
    local top_high_roller_spent
    top_high_roller_spent=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            SELECT discord_id, COUNT(*) as purchase_count, SUM(purchase_price) as total_spent
            FROM high_roller_purchases
            WHERE guild_id = '$GUILD_ID' AND purchased_at >= '$START_DATE' AND purchased_at < '$END_DATE'
            GROUP BY discord_id ORDER BY total_spent DESC LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"top_spenders\": ${top_high_roller_spent:-null}"

    json_output+="},"

    # LoL Summoner Stats
    json_output+="\"lol\": {"

    # Top 3 Most Profitable Summoners - array with wager_count, total_wagered, net_profit
    local lol_most_profitable
    lol_most_profitable=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            WITH lol_games AS (
                SELECT gw.id as wager_id, TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
                FROM group_wagers gw
                WHERE gw.guild_id = '$GUILD_ID' AND gw.external_system = 'league_of_legends'
                    AND gw.state = 'resolved' AND gw.created_at >= '$START_DATE' AND gw.created_at < '$END_DATE'
            ),
            bettor_outcomes AS (
                SELECT lg.summoner_name, lg.wager_id, p.amount as bet_amount,
                    COALESCE(p.payout_amount, 0) - p.amount as net_result
                FROM lol_games lg
                JOIN group_wager_participants p ON p.group_wager_id = lg.wager_id
                WHERE lg.summoner_name IS NOT NULL AND lg.summoner_name != ''
            )
            SELECT summoner_name as summoner, COUNT(DISTINCT wager_id) as wager_count,
                SUM(bet_amount) as total_wagered, SUM(net_result) as net_profit
            FROM bettor_outcomes
            GROUP BY summoner_name
            HAVING COUNT(DISTINCT wager_id) > 0
            ORDER BY net_profit DESC LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"most_profitable_summoners\": ${lol_most_profitable:-null},"

    # Top 3 Least Profitable Summoners - array with wager_count, total_wagered, net_profit
    local lol_least_profitable
    lol_least_profitable=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            WITH lol_games AS (
                SELECT gw.id as wager_id, TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
                FROM group_wagers gw
                WHERE gw.guild_id = '$GUILD_ID' AND gw.external_system = 'league_of_legends'
                    AND gw.state = 'resolved' AND gw.created_at >= '$START_DATE' AND gw.created_at < '$END_DATE'
            ),
            bettor_outcomes AS (
                SELECT lg.summoner_name, lg.wager_id, p.amount as bet_amount,
                    COALESCE(p.payout_amount, 0) - p.amount as net_result
                FROM lol_games lg
                JOIN group_wager_participants p ON p.group_wager_id = lg.wager_id
                WHERE lg.summoner_name IS NOT NULL AND lg.summoner_name != ''
            )
            SELECT summoner_name as summoner, COUNT(DISTINCT wager_id) as wager_count,
                SUM(bet_amount) as total_wagered, SUM(net_result) as net_profit
            FROM bettor_outcomes
            GROUP BY summoner_name
            HAVING COUNT(DISTINCT wager_id) > 0
            ORDER BY net_profit ASC LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"least_profitable_summoners\": ${lol_least_profitable:-null},"

    json_output+="\"most_bet_on_summoner\": $(run_stat_query "most_bet_on_summoner" "
        WITH lol_games AS (
            SELECT gw.id as wager_id, TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
            FROM group_wagers gw
            WHERE gw.guild_id = \$1 AND gw.external_system = 'league_of_legends'
                AND gw.state = 'resolved' AND gw.created_at >= \$2 AND gw.created_at < \$3
        )
        SELECT lg.summoner_name as summoner, COUNT(DISTINCT lg.wager_id) as wager_count, SUM(p.amount) as total_wagered
        FROM lol_games lg
        JOIN group_wager_participants p ON p.group_wager_id = lg.wager_id
        WHERE lg.summoner_name IS NOT NULL AND lg.summoner_name != ''
        GROUP BY lg.summoner_name
        ORDER BY total_wagered DESC LIMIT 1
    "),"

    json_output+="\"total_amount_wagered\": $(run_stat_query "total_amount_wagered" "
        SELECT COALESCE(SUM(p.amount), 0) as amount
        FROM group_wager_participants p
        JOIN group_wagers gw ON gw.id = p.group_wager_id
        WHERE gw.guild_id = \$1 AND gw.external_system = 'league_of_legends'
            AND gw.created_at >= \$2 AND gw.created_at < \$3
    ")"

    json_output+="},"

    # TFT Summoner Stats
    json_output+="\"tft\": {"

    # Top 3 Most Profitable Summoners - array with wager_count, total_wagered, net_profit
    local tft_most_profitable
    tft_most_profitable=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            WITH tft_games AS (
                SELECT gw.id as wager_id, TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
                FROM group_wagers gw
                WHERE gw.guild_id = '$GUILD_ID' AND gw.external_system = 'teamfight_tactics'
                    AND gw.state = 'resolved' AND gw.created_at >= '$START_DATE' AND gw.created_at < '$END_DATE'
            ),
            bettor_outcomes AS (
                SELECT tg.summoner_name, tg.wager_id, p.amount as bet_amount,
                    COALESCE(p.payout_amount, 0) - p.amount as net_result
                FROM tft_games tg
                JOIN group_wager_participants p ON p.group_wager_id = tg.wager_id
                WHERE tg.summoner_name IS NOT NULL AND tg.summoner_name != ''
            )
            SELECT summoner_name as summoner, COUNT(DISTINCT wager_id) as wager_count,
                SUM(bet_amount) as total_wagered, SUM(net_result) as net_profit
            FROM bettor_outcomes
            GROUP BY summoner_name
            HAVING COUNT(DISTINCT wager_id) > 0
            ORDER BY net_profit DESC LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"most_profitable_summoners\": ${tft_most_profitable:-null},"

    # Top 3 Least Profitable Summoners - array with wager_count, total_wagered, net_profit
    local tft_least_profitable
    tft_least_profitable=$(psql "$DATABASE_URL" -t -c "
        SELECT json_agg(row_to_json(t))
        FROM (
            WITH tft_games AS (
                SELECT gw.id as wager_id, TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
                FROM group_wagers gw
                WHERE gw.guild_id = '$GUILD_ID' AND gw.external_system = 'teamfight_tactics'
                    AND gw.state = 'resolved' AND gw.created_at >= '$START_DATE' AND gw.created_at < '$END_DATE'
            ),
            bettor_outcomes AS (
                SELECT tg.summoner_name, tg.wager_id, p.amount as bet_amount,
                    COALESCE(p.payout_amount, 0) - p.amount as net_result
                FROM tft_games tg
                JOIN group_wager_participants p ON p.group_wager_id = tg.wager_id
                WHERE tg.summoner_name IS NOT NULL AND tg.summoner_name != ''
            )
            SELECT summoner_name as summoner, COUNT(DISTINCT wager_id) as wager_count,
                SUM(bet_amount) as total_wagered, SUM(net_result) as net_profit
            FROM bettor_outcomes
            GROUP BY summoner_name
            HAVING COUNT(DISTINCT wager_id) > 0
            ORDER BY net_profit ASC LIMIT 3
        ) t;
    " 2>/dev/null | tr -d '[:space:]')
    json_output+="\"least_profitable_summoners\": ${tft_least_profitable:-null},"

    json_output+="\"most_bet_on_summoner\": $(run_stat_query "most_bet_on_summoner" "
        WITH tft_games AS (
            SELECT gw.id as wager_id, TRIM(SPLIT_PART(gw.condition, ' - **', 1)) as summoner_name
            FROM group_wagers gw
            WHERE gw.guild_id = \$1 AND gw.external_system = 'teamfight_tactics'
                AND gw.state = 'resolved' AND gw.created_at >= \$2 AND gw.created_at < \$3
        )
        SELECT tg.summoner_name as summoner, COUNT(DISTINCT tg.wager_id) as wager_count, SUM(p.amount) as total_wagered
        FROM tft_games tg
        JOIN group_wager_participants p ON p.group_wager_id = tg.wager_id
        WHERE tg.summoner_name IS NOT NULL AND tg.summoner_name != ''
        GROUP BY tg.summoner_name
        ORDER BY total_wagered DESC LIMIT 1
    "),"

    json_output+="\"total_amount_wagered\": $(run_stat_query "total_amount_wagered" "
        SELECT COALESCE(SUM(p.amount), 0) as amount
        FROM group_wager_participants p
        JOIN group_wagers gw ON gw.id = p.group_wager_id
        WHERE gw.guild_id = \$1 AND gw.external_system = 'teamfight_tactics'
            AND gw.created_at >= \$2 AND gw.created_at < \$3
    ")"

    json_output+="}"

    # Close main object
    json_output+="}"

    echo "$json_output"
}

# Run and output
collect_stats | jq .
