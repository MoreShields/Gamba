-- Remove redundant max_guesses column from wordle_completions table
-- Wordle always has exactly 6 maximum guesses
ALTER TABLE wordle_completions DROP COLUMN max_guesses;