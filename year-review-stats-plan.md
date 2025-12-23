# Year in Review Stats Plan

## Overview
Create a dataset and queries to generate fun, shareable statistics for a "Year in Review" graphic showing gambling participation and outcomes over the past year.

---

## Data Sources

| Table | Purpose |
|-------|---------|
| `bets` | House gambling (slots, probability-based games) |
| `wagers` | Head-to-head wagers between users |
| `group_wagers` + `group_wager_participants` | Pool and house group wagers |
| `group_wagers` (external_system='league_of_legends') | LoL match betting |
| `group_wagers` (external_system='teamfight_tactics') | TFT match betting |
| `balance_history` | Complete financial transaction audit trail |
| `wordle_completions` | Daily Wordle participation and streaks |
| `high_roller_purchases` | Status/role purchases |

---

## Stat Categories

### 1. Volume Leaders (Who Gambled the Most)

| Stat | Query Logic |
|------|-------------|
| **Most Bets Placed** | COUNT from `bets` by discord_id |
| **Highest Total Amount Wagered** | SUM(amount) from `bets` by discord_id |
| **Most Wagers Proposed** | COUNT from `wagers` WHERE proposer_discord_id |
| **Most Wagers Accepted** | COUNT from `wagers` WHERE target_discord_id AND state != 'declined' |
| **Most Group Wager Participations** | COUNT from `group_wager_participants` by discord_id |
| **Most Active Gambling Days** | COUNT(DISTINCT DATE(created_at)) from `bets` |

### 2. Win/Loss Records

| Stat | Query Logic |
|------|-------------|
| **Most Bet Wins** | COUNT from `bets` WHERE won = true |
| **Most Bet Losses** | COUNT from `bets` WHERE won = false |
| **Best Bet Win Rate** | (wins / total) WHERE total >= min_threshold |
| **Worst Bet Win Rate** | (wins / total) WHERE total >= min_threshold |
| **Biggest Single Win** | MAX(win_amount) from `bets` WHERE won = true |
| **Biggest Single Loss** | MAX(amount) from `bets` WHERE won = false |
| **Most H2H Wager Wins** | COUNT from `wagers` WHERE winner_discord_id |
| **Most H2H Wager Losses** | COUNT WHERE participant AND winner != self |
| **Most Group Wager Wins** | COUNT WHERE payout_amount > amount bet |
| **Longest Winning Streak** | Consecutive wins in `bets` (window function) |
| **Longest Losing Streak** | Consecutive losses in `bets` |

### 3. Effective Odds Analysis

| Stat | Query Logic |
|------|-------------|
| **Best Luck (Bets)** | (actual_win_rate / expected_win_rate) - players who beat the odds |
| **Worst Luck (Bets)** | Players who significantly underperformed vs probability |
| **Luckiest at High-Risk Bets** | Best performance on bets with win_probability < 0.3 |
| **Luckiest at Safe Bets** | Best performance on bets with win_probability > 0.7 |
| **Most Profitable Risk-Taker** | Net profit on low-probability bets |

### 4. Financial Impact

| Stat | Query Logic |
|------|-------------|
| **Biggest Net Winner** | SUM of all transaction changes (bet_win - bet_loss + wager_win - wager_loss + ...) |
| **Biggest Net Loser** | Inverse of above |
| **Biggest Comeback** | Largest improvement from lowest balance to highest |
| **Biggest Fall** | Largest drop from highest balance to lowest |
| **Most Volatile Balance** | Standard deviation of balance over time |
| **Most Transfers Sent** | COUNT of transfer_out transactions |
| **Most Generous** | SUM(amount) of transfers sent |

### 5. Wordle Stats

| Stat | Query Logic |
|------|-------------|
| **Longest Wordle Streak** | Consecutive days with completion |
| **Most Wordle Completions** | COUNT from `wordle_completions` |
| **Best Average Guess Count** | AVG(guess_count) with min completions |
| **Most 1-Guess Wins** | COUNT WHERE guess_count = 1 |
| **Most Wordle Rewards Earned** | SUM from balance_history WHERE transaction_type = 'wordle_reward' |

### 6. Social & Community Stats

| Stat | Query Logic |
|------|-------------|
| **Most Popular Wager Target** | Most times as target_discord_id |
| **Most Trusted Resolver** | COUNT from `group_wagers` WHERE resolver_discord_id |
| **Most Group Wagers Created** | COUNT from `group_wagers` WHERE creator_discord_id |
| **Biggest Pool Creator** | MAX(total_pot) for created group wagers |
| **Most Wager Votes Cast** | COUNT from `wager_votes` |

### 7. High Roller Stats

| Stat | Query Logic |
|------|-------------|
| **Most High Roller Purchases** | COUNT from `high_roller_purchases` |
| **Highest Total Spent on High Roller** | SUM(purchase_price) |
| **Current High Roller** | Most recent purchase per guild |

### 8. Fun/Quirky Stats

| Stat | Query Logic |
|------|-------------|
| **Night Owl** | Most bets placed between midnight-4am |
| **Early Bird** | Most bets placed between 5am-8am |
| **Weekend Warrior** | Highest % of bets on Saturday/Sunday |
| **Lucky Number Seeker** | Most bets at specific probability thresholds (e.g., 0.5, 0.25) |
| **The Doubler** | Most times betting immediately after a loss |
| **Patience Pays** | Longest gap between bets |
| **Speed Demon** | Shortest average time between bets |
| **Jinxed Favorite** | Most times being the majority vote but losing |
| **Underdog Champion** | Most times winning as minority vote |
| **The Nemesis** | Pair of users with most head-to-head wagers against each other |
| **Rivalry Winner** | Who leads in head-to-head matchups |

### 9. League of Legends Summoner Stats

LoL games are tracked as `group_wagers` with `external_system = 'league_of_legends'`.
Summoner name is extracted from the `condition` field using regex: `\[([^\]]+)\]`.

| Stat | Query Logic |
|------|-------------|
| **Most Winning Summoner (Win Rate)** | Highest win rate from resolved LoL wagers (min games threshold) |
| **Least Winning Summoner (Win Rate)** | Lowest win rate from resolved LoL wagers |
| **Most Profitable Summoner** | Summoner with highest total net winnings for bettors |
| **Most Costly Summoner** | Summoner with highest total net losses for bettors |
| **Most Bet-On Summoner** | Summoner with most total amount wagered across all games |
| **Most Games Tracked** | Summoner with most resolved wagers |
| **Biggest Single Game Pot** | Largest total_pot for a single LoL wager |

### 10. TFT Summoner Stats

TFT games are tracked as `group_wagers` with `external_system = 'teamfight_tactics'`.
Placement options: "1-2", "3-4", "5-6", "7-8" (regular) or "1", "2", "3", "4" (Double Up).

| Stat | Query Logic |
|------|-------------|
| **Best Average Placement** | Lowest avg placement (map winning_option to placement midpoint) |
| **Worst Average Placement** | Highest avg placement |
| **Most Top-4 Finishes** | COUNT where winning_option IN ('1-2', '3-4', '1', '2') |
| **Most Bottom-4 Finishes** | COUNT where winning_option IN ('5-6', '7-8', '3', '4') |
| **Most TFT Games Tracked** | Summoner with most resolved TFT wagers |
| **Most Profitable TFT Summoner** | Highest net winnings for bettors |
| **Most Bet-On TFT Summoner** | Highest total wagered amount |

### 11. Aggregate Platform Stats

| Stat | Query Logic |
|------|-------------|
| **Total Bets Placed** | COUNT(*) from `bets` |
| **Total Amount Wagered** | SUM(amount) from `bets` |
| **Total Wagers Created** | COUNT(*) from `wagers` |
| **Total Group Wagers** | COUNT(*) from `group_wagers` |
| **Active Users** | COUNT(DISTINCT discord_id) across all activity |
| **Busiest Day** | DATE with most transactions |
| **Busiest Hour** | HOUR with most transactions |
| **House Edge Collected** | Total bet_loss - bet_win |
| **Most Popular Bet Probability** | MODE of win_probability |
| **Average Bet Size** | AVG(amount) from `bets` |

---

## Implementation Plan

### Phase 1: SQL Query Development
Create a SQL file with all year-in-review queries, parameterized by:
- `guild_id` (required)
- `start_date` (default: Jan 1 of current year)
- `end_date` (default: Dec 31 of current year)
- `min_participation_threshold` (to filter out inactive users)

### Phase 2: Data Aggregation Script
Create a script that:
1. Connects to the database
2. Executes all queries
3. Assembles results into a structured JSON output
4. Includes user discord_ids for username resolution

### Phase 3: Output Format
Generate a JSON structure like:
```json
{
  "guild_id": 123456789,
  "year": 2024,
  "generated_at": "2024-12-23T00:00:00Z",
  "stats": {
    "volume_leaders": {
      "most_bets_placed": { "discord_id": "...", "count": 1234 },
      ...
    },
    "win_loss_records": { ... },
    "effective_odds": { ... },
    ...
  },
  "platform_totals": { ... }
}
```

---

## Query Examples

### Most Bets Placed
```sql
SELECT
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
```

### Best Luck vs Odds (Bets)
```sql
WITH user_stats AS (
    SELECT
        discord_id,
        COUNT(*) as total_bets,
        AVG(win_probability) as avg_expected_win_rate,
        AVG(CASE WHEN won THEN 1.0 ELSE 0.0 END) as actual_win_rate
    FROM bets
    WHERE guild_id = $1
      AND created_at >= $2
      AND created_at < $3
    GROUP BY discord_id
    HAVING COUNT(*) >= $4  -- min threshold
)
SELECT
    discord_id,
    total_bets,
    ROUND(avg_expected_win_rate * 100, 2) as expected_win_rate_pct,
    ROUND(actual_win_rate * 100, 2) as actual_win_rate_pct,
    ROUND((actual_win_rate - avg_expected_win_rate) * 100, 2) as luck_factor_pct
FROM user_stats
ORDER BY (actual_win_rate - avg_expected_win_rate) DESC
LIMIT 1;
```

### Longest Winning Streak
```sql
WITH bet_results AS (
    SELECT
        discord_id,
        created_at,
        won,
        ROW_NUMBER() OVER (PARTITION BY discord_id ORDER BY created_at) -
        ROW_NUMBER() OVER (PARTITION BY discord_id, won ORDER BY created_at) as streak_group
    FROM bets
    WHERE guild_id = $1
      AND created_at >= $2
      AND created_at < $3
),
streaks AS (
    SELECT
        discord_id,
        won,
        COUNT(*) as streak_length,
        MIN(created_at) as streak_start,
        MAX(created_at) as streak_end
    FROM bet_results
    GROUP BY discord_id, won, streak_group
)
SELECT
    discord_id,
    streak_length as longest_winning_streak,
    streak_start,
    streak_end
FROM streaks
WHERE won = true
ORDER BY streak_length DESC
LIMIT 1;
```

### The Nemesis (Most H2H Matchups)
```sql
WITH matchups AS (
    SELECT
        LEAST(proposer_discord_id, target_discord_id) as user1,
        GREATEST(proposer_discord_id, target_discord_id) as user2,
        COUNT(*) as wager_count,
        SUM(CASE WHEN winner_discord_id = LEAST(proposer_discord_id, target_discord_id) THEN 1 ELSE 0 END) as user1_wins,
        SUM(CASE WHEN winner_discord_id = GREATEST(proposer_discord_id, target_discord_id) THEN 1 ELSE 0 END) as user2_wins
    FROM wagers
    WHERE guild_id = $1
      AND state = 'resolved'
      AND created_at >= $2
      AND created_at < $3
    GROUP BY
        LEAST(proposer_discord_id, target_discord_id),
        GREATEST(proposer_discord_id, target_discord_id)
)
SELECT
    user1,
    user2,
    wager_count as total_matchups,
    user1_wins,
    user2_wins,
    CASE WHEN user1_wins > user2_wins THEN user1
         WHEN user2_wins > user1_wins THEN user2
         ELSE NULL END as rivalry_leader
FROM matchups
ORDER BY wager_count DESC
LIMIT 1;
```

### Biggest Comeback
```sql
WITH balance_timeline AS (
    SELECT
        discord_id,
        balance_after,
        created_at,
        MIN(balance_after) OVER (PARTITION BY discord_id ORDER BY created_at) as running_min
    FROM balance_history
    WHERE guild_id = $1
      AND created_at >= $2
      AND created_at < $3
)
SELECT
    discord_id,
    MIN(running_min) as lowest_point,
    MAX(balance_after) as highest_after_low,
    MAX(balance_after) - MIN(running_min) as comeback_amount
FROM balance_timeline
GROUP BY discord_id
ORDER BY comeback_amount DESC
LIMIT 1;
```

### LoL Summoner Win Rate (Best/Worst)
```sql
WITH lol_games AS (
    SELECT
        gw.id,
        gw.condition,
        gw.winning_option_id,
        gw.total_pot,
        -- Extract summoner name from condition: "[SummonerName](url) - ..."
        SUBSTRING(gw.condition FROM '\[([^\]]+)\]') as summoner_name,
        wo.option_text as winning_option
    FROM group_wagers gw
    JOIN group_wager_options wo ON wo.id = gw.winning_option_id
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
    GROUP BY summoner_name
    HAVING COUNT(*) >= $4  -- min games threshold
)
SELECT
    summoner_name,
    total_games,
    wins,
    losses,
    ROUND(wins::decimal / total_games * 100, 2) as win_rate_pct,
    total_wagered_on
FROM summoner_stats
ORDER BY win_rate_pct DESC  -- Change to ASC for worst
LIMIT 1;
```

### Most Profitable/Costly LoL Summoner (for Bettors)
```sql
WITH lol_games AS (
    SELECT
        gw.id as wager_id,
        SUBSTRING(gw.condition FROM '\[([^\]]+)\]') as summoner_name,
        gw.winning_option_id
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
)
SELECT
    summoner_name,
    COUNT(DISTINCT discord_id) as unique_bettors,
    SUM(bet_amount) as total_wagered,
    SUM(payout) as total_payouts,
    SUM(net_result) as net_bettor_profit  -- positive = bettors made money overall
FROM bettor_outcomes
GROUP BY summoner_name
ORDER BY net_bettor_profit DESC  -- Change to ASC for most costly
LIMIT 1;
```

### TFT Average Placement (Best/Worst)
```sql
WITH tft_games AS (
    SELECT
        gw.id,
        SUBSTRING(gw.condition FROM '\[([^\]]+)\]') as summoner_name,
        wo.option_text as winning_option,
        -- Map placement ranges to numeric values
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
    WHERE placement_value IS NOT NULL
    GROUP BY summoner_name
    HAVING COUNT(*) >= $4  -- min games threshold
)
SELECT
    summoner_name,
    total_games,
    ROUND(avg_placement::decimal, 2) as avg_placement,
    top_4_count,
    bottom_4_count,
    ROUND(top_4_count::decimal / total_games * 100, 2) as top_4_rate_pct
FROM summoner_stats
ORDER BY avg_placement ASC  -- Change to DESC for worst
LIMIT 1;
```

---

## Minimum Participation Thresholds

To ensure meaningful stats, apply thresholds:
- **Bet stats**: Minimum 10 bets
- **Win rate stats**: Minimum 25 bets
- **Luck factor stats**: Minimum 50 bets
- **Wager stats**: Minimum 3 wagers
- **Wordle stats**: Minimum 10 completions
- **LoL summoner stats**: Minimum 5 games
- **TFT summoner stats**: Minimum 5 games

---

## File Structure

```
discord-client/
├── database/
│   └── queries/
│       └── year_review.sql          # All SQL queries
├── src/
│   └── commands/
│       └── yearReview.ts             # Command handler
│   └── services/
│       └── yearReviewService.ts      # Business logic
│   └── types/
│       └── yearReview.ts             # TypeScript interfaces
```

---

## Next Steps

1. [ ] Review and approve this plan
2. [ ] Create the SQL queries file with all year-in-review queries
3. [ ] Create TypeScript service to execute queries and aggregate results
4. [ ] Create Discord command to generate and display stats
5. [ ] Test with sample data
6. [ ] Consider export format for graphic generation (JSON/CSV)
