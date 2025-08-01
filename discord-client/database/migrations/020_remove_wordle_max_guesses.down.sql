-- Re-add max_guesses column to wordle_completions table
-- Default to 6 since Wordle always has 6 maximum guesses
ALTER TABLE wordle_completions ADD COLUMN max_guesses INT NOT NULL DEFAULT 6 CHECK (max_guesses BETWEEN 1 AND 6);