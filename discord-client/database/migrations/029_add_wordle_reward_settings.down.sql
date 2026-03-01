ALTER TABLE guild_settings
    DROP COLUMN IF EXISTS wordle_reward_score_1,
    DROP COLUMN IF EXISTS wordle_reward_score_2_4,
    DROP COLUMN IF EXISTS wordle_reward_score_5_6;
