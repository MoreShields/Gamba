package observability

// Metric name prefixes
const (
	MetricPrefix = "discord_client"
)

// Metric names
const (
	// Discord metrics
	MessagesReadTotal = MetricPrefix + ".messages.read_total"

	// Wager metrics
	WagersActive = MetricPrefix + ".wagers.active"

	// NATS metrics
	NATSMessagesReceivedTotal  = MetricPrefix + ".nats.messages_received_total"
	NATSMessagesPublishedTotal = MetricPrefix + ".nats.messages_published_total"

	// Balance metrics
	BalanceTransactionsTotal = MetricPrefix + ".balance.transactions_total"

	// Database metrics
	DatabaseQueriesTotal   = MetricPrefix + ".database.queries_total"
	DatabaseQueryDuration = MetricPrefix + ".database.query_duration"
)

// Label keys
const (
	// Common labels
	LabelType      = "type"
	LabelEventType = "event_type"

	// Database labels
	LabelRepository = "repository"
	LabelMethod     = "method"

	// Error labels
	LabelErrorType = "error_type"
)

// Message types for Discord
const (
	MessageTypeCommand     = "command"
	MessageTypeInteraction = "interaction"
	MessageTypeMessage     = "message"
)

// Wager types
const (
	WagerTypeLoL   = "lol"
	WagerTypeTFT   = "tft"
	WagerTypeGroup = "group"
)

// Balance transaction types
const (
	TransactionTypeWager        = "wager"
	TransactionTypeBet          = "bet"
	TransactionTypeTransfer     = "transfer"
	TransactionTypeWordleReward = "wordle_reward"
	TransactionTypeDailyAward   = "daily_award"
	TransactionTypeHighRoller   = "high_roller"
	TransactionTypeRefund       = "refund"
)