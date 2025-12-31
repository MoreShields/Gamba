package betting

import (
	"sync"
	"time"
)

// BetSession stores temporary betting state for a user
type BetSession struct {
	UserID         int64
	MessageID      string
	ChannelID      string
	LastOdds       float64
	LastAmount     int64
	CurrentBalance int64
	Timestamp      time.Time
	// Session tracking fields
	StartingBalance int64 // Balance when session began
	SessionPnL      int64 // Running total P&L for this session
	BetCount        int   // Number of bets placed in session
}

var (
	betSessions   = make(map[int64]*BetSession) // userID → session
	messageToUser = make(map[string]int64)      // messageID → userID (reverse lookup)
	betSessionsMu sync.RWMutex
)

// cleanupSessions removes sessions older than 1 hour
func cleanupSessions() {
	betSessionsMu.Lock()
	defer betSessionsMu.Unlock()

	now := time.Now()
	for userID, session := range betSessions {
		if now.Sub(session.Timestamp) > time.Hour {
			delete(messageToUser, session.MessageID)
			delete(betSessions, userID)
		}
	}
}

// getBetSession retrieves a bet session for a user
func getBetSession(userID int64) *BetSession {
	betSessionsMu.RLock()
	defer betSessionsMu.RUnlock()
	return betSessions[userID]
}

// getSessionOwnerByMessageID returns the owner's userID for a message, or 0 if not found
func getSessionOwnerByMessageID(messageID string) int64 {
	betSessionsMu.RLock()
	defer betSessionsMu.RUnlock()
	return messageToUser[messageID]
}

// createBetSession creates a new betting session
func createBetSession(userID int64, messageID, channelID string, balance int64) {
	betSessionsMu.Lock()
	defer betSessionsMu.Unlock()

	// Clean up old message mapping if user had previous session
	if oldSession, exists := betSessions[userID]; exists {
		delete(messageToUser, oldSession.MessageID)
	}

	betSessions[userID] = &BetSession{
		UserID:          userID,
		MessageID:       messageID,
		ChannelID:       channelID,
		CurrentBalance:  balance,
		StartingBalance: balance,
		SessionPnL:      0,
		BetCount:        0,
		Timestamp:       time.Now(),
	}
	messageToUser[messageID] = userID
}

// updateBetSession updates an existing betting session
func updateBetSession(userID int64, odds float64, amount int64) {
	betSessionsMu.Lock()
	defer betSessionsMu.Unlock()
	if session, ok := betSessions[userID]; ok {
		session.LastOdds = odds
		session.LastAmount = amount
		session.Timestamp = time.Now()
	}
}

// updateSessionBalance updates the balance and P&L for a session
func updateSessionBalance(userID int64, newBalance int64, betPlaced bool) {
	betSessionsMu.Lock()
	defer betSessionsMu.Unlock()
	if session, ok := betSessions[userID]; ok {
		session.CurrentBalance = newBalance
		session.SessionPnL = newBalance - session.StartingBalance
		if betPlaced {
			session.BetCount++
		}
		session.Timestamp = time.Now()
	}
}

// deleteBetSession removes a betting session
func deleteBetSession(userID int64) {
	betSessionsMu.Lock()
	defer betSessionsMu.Unlock()
	if session, exists := betSessions[userID]; exists {
		delete(messageToUser, session.MessageID)
	}
	delete(betSessions, userID)
}
