@startuml Database Schema

!theme plain

' Users table
entity "users" {
  * id : SERIAL <<PK>>
  --
  * discord_id : TEXT <<UNIQUE>>
  * username : TEXT
  * balance : BIGINT
  * created_at : TIMESTAMP
  * updated_at : TIMESTAMP
}

' Balance History table
entity "balance_history" {
  * id : SERIAL <<PK>>
  --
  * user_id : INTEGER <<FK>>
  * amount : BIGINT
  * transaction_type : TEXT
  * description : TEXT
  * bet_id : INTEGER <<FK>>
  * wager_id : INTEGER <<FK>>
  * group_wager_id : INTEGER <<FK>>
  * created_at : TIMESTAMP
}

' Bets table
entity "bets" {
  * id : SERIAL <<PK>>
  --
  * user_id : INTEGER <<FK>>
  * amount : BIGINT
  * odds : DECIMAL
  * won : BOOLEAN
  * payout : BIGINT
  * created_at : TIMESTAMP
}

' Wagers table
entity "wagers" {
  * id : SERIAL <<PK>>
  --
  * creator_id : INTEGER <<FK>>
  * target_id : INTEGER <<FK>>
  * amount : BIGINT
  * condition : TEXT
  * state : TEXT
  * winner_id : INTEGER <<FK>>
  * created_at : TIMESTAMP
  * updated_at : TIMESTAMP
}

' Wager Votes table
entity "wager_votes" {
  * id : SERIAL <<PK>>
  --
  * wager_id : INTEGER <<FK>>
  * voter_id : INTEGER <<FK>>
  * vote_for_id : INTEGER <<FK>>
  * created_at : TIMESTAMP
  * updated_at : TIMESTAMP
}

' Group Wagers table
entity "group_wagers" {
  * id : SERIAL <<PK>>
  --
  * creator_id : INTEGER <<FK>>
  * condition : TEXT
  * state : TEXT
  * min_participants : INTEGER
  * max_participants : INTEGER
  * entry_amount : BIGINT
  * winner_id : INTEGER <<FK>>
  * created_at : TIMESTAMP
  * updated_at : TIMESTAMP
}

' Group Wager Participants table
entity "group_wager_participants" {
  * id : SERIAL <<PK>>
  --
  * group_wager_id : INTEGER <<FK>>
  * user_id : INTEGER <<FK>>
  * joined_at : TIMESTAMP
}

' Group Wager Votes table
entity "group_wager_votes" {
  * id : SERIAL <<PK>>
  --
  * group_wager_id : INTEGER <<FK>>
  * voter_id : INTEGER <<FK>>
  * vote_for_id : INTEGER <<FK>>
  * created_at : TIMESTAMP
  * updated_at : TIMESTAMP
}

' Interest Runs table
entity "interest_runs" {
  * id : SERIAL <<PK>>
  --
  * run_date : DATE <<UNIQUE>>
  * interest_rate : DECIMAL
  * total_distributed : BIGINT
  * users_affected : INTEGER
  * created_at : TIMESTAMP
}

' Relationships
users ||--o{ balance_history : "user_id"
users ||--o{ bets : "user_id"
users ||--o{ wagers : "creator_id"
users ||--o{ wagers : "target_id"
users ||--o{ wagers : "winner_id"
users ||--o{ wager_votes : "voter_id"
users ||--o{ wager_votes : "vote_for_id"
users ||--o{ group_wagers : "creator_id"
users ||--o{ group_wagers : "winner_id"
users ||--o{ group_wager_participants : "user_id"
users ||--o{ group_wager_votes : "voter_id"
users ||--o{ group_wager_votes : "vote_for_id"

bets ||--o{ balance_history : "bet_id"
wagers ||--o{ balance_history : "wager_id"
wagers ||--o{ wager_votes : "wager_id"
group_wagers ||--o{ balance_history : "group_wager_id"
group_wagers ||--o{ group_wager_participants : "group_wager_id"
group_wagers ||--o{ group_wager_votes : "group_wager_id"

@enduml