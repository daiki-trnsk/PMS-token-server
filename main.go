package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	lkauth "github.com/livekit/protocol/auth"
)

type TokenRequest struct {
	Identity string `json:"identity"`
	Room     string `json:"room"`
}

type TokenResponse struct {
	Token    string `json:"token"`
	URL      string `json:"url"`
	Room     string `json:"room"`
	Identity string `json:"identity"`
	Name     string `json:"name"`
}

func main() {
	_ = godotenv.Load()

	apiKey := os.Getenv("LIVEKIT_API_KEY")
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")
	livekitURL := os.Getenv("LIVEKIT_URL")

	if apiKey == "" || apiSecret == "" || livekitURL == "" {
		log.Fatal("LIVEKIT_API_KEY, LIVEKIT_API_SECRET, LIVEKIT_URL are required")
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	http.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req TokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.Identity == "" {
			http.Error(w, "identity is required", http.StatusBadRequest)
			return
		}
		if req.Room == "" {
			http.Error(w, "room is required", http.StatusBadRequest)
			return
		}

		allowedRooms := map[string]map[string]bool{
			"participant-a":    {"room-a": true},
			"participant-b":    {"room-a": true},
			"participant-c":    {"room-b": true},
			"participant-d":    {"room-b": true},
			"subject-01":       {"room-a": true, "room-b": true},
			"pms-agent-room-a": {"room-a": true},
			"pms-agent-room-b": {"room-b": true},
		}

		allowedForIdentity, ok := allowedRooms[req.Identity]
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !allowedForIdentity[req.Room] {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		at := lkauth.NewAccessToken(apiKey, apiSecret)
		at.SetIdentity(req.Identity)
		at.SetName(req.Identity)
		at.SetValidFor(2 * time.Hour)

		canPublish := true
		canSubscribe := true

		grant := &lkauth.VideoGrant{
			RoomJoin:     true,
			Room:         req.Room,
			CanPublish:   &canPublish,
			CanSubscribe: &canSubscribe,
		}
		at.SetVideoGrant(grant)

		token, err := at.ToJWT()
		if err != nil {
			http.Error(w, "failed to create token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			Token:    token,
			URL:      livekitURL,
			Room:     req.Room,
			Identity: req.Identity,
			Name:     req.Identity,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("server running on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
