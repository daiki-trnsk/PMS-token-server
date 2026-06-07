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

		// validate identity
		if req.Identity == "" {
			http.Error(w, "identity is required", http.StatusBadRequest)
			return
		}

		// allowed identities and display names
		const fixedRoom = "experiment-001"
		allowed := map[string]string{
			"subject-01": "同時参加者",
			"sakura-a":   "サクラA",
			"sakura-b":   "サクラB",
			"sakura-c":   "サクラC",
			"sakura-d":   "サクラD",
		}

		displayName, ok := allowed[req.Identity]
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		at := lkauth.NewAccessToken(apiKey, apiSecret)
		at.SetIdentity(req.Identity)
		at.SetName(displayName)
		at.SetValidFor(2 * time.Hour)

		canPublish := true
		canSubscribe := true

		grant := &lkauth.VideoGrant{
			RoomJoin:     true,
			Room:         fixedRoom,
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
			Room:     fixedRoom,
			Identity: req.Identity,
			Name:     displayName,
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("server running on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
