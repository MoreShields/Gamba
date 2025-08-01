package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// DebugCommand represents a debug command sent via HTTP
type DebugCommand struct {
	Action  string            `json:"action"`
	Params  map[string]string `json:"params"`
}

// DebugResponse represents the response from a debug command
type DebugResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// GuildInfo represents basic guild information
type GuildInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// StartDebugAPI starts an internal HTTP API for debug commands
func (b *Bot) StartDebugAPI(port int) error {
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Get guilds endpoint
	mux.HandleFunc("/debug/guilds", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		guilds := b.GetGuilds()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DebugResponse{
			Success: true,
			Data:    guilds,
		})
	})
	
	// Debug command endpoint
	mux.HandleFunc("/debug/command", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		var cmd DebugCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			respondWithError(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// Handle different debug actions
		switch cmd.Action {
		case "replay":
			channelID := cmd.Params["channel_id"]
			messageID := cmd.Params["message_id"]
			
			if channelID == "" || messageID == "" {
				respondWithError(w, "Missing channel_id or message_id", http.StatusBadRequest)
				return
			}
			
			if err := b.ReplayMessage(channelID, messageID); err != nil {
				respondWithError(w, fmt.Sprintf("Failed to replay message: %v", err), http.StatusInternalServerError)
				return
			}
			
			respondWithSuccess(w, "Message replayed successfully")
			
		default:
			respondWithError(w, fmt.Sprintf("Unknown action: %s", cmd.Action), http.StatusBadRequest)
		}
	})
	
	// Start server in background
	server := &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	
	go func() {
		log.Infof("Debug API listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Debug API server error: %v", err)
		}
	}()
	
	// Store server reference if we need to shut it down later
	// b.debugServer = server
	
	return nil
}

func respondWithSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DebugResponse{
		Success: true,
		Message: message,
	})
}

func respondWithError(w http.ResponseWriter, error string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(DebugResponse{
		Success: false,
		Error:   error,
	})
}