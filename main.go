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
	Room     string `json:"room"`
	Identity string `json:"identity"`
	Name     string `json:"name"`
}

type TokenResponse struct {
	Token string `json:"token"`
	URL   string `json:"url"`
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

		if req.Room == "" || req.Identity == "" {
			http.Error(w, "room and identity are required", http.StatusBadRequest)
			return
		}

		at := lkauth.NewAccessToken(apiKey, apiSecret)
		at.SetIdentity(req.Identity)
		at.SetName(req.Name)
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
			Token: token,
			URL:   livekitURL,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("server running on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
