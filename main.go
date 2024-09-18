package main

import (
	"TimeTrackerApplication/internal/database"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type apiConfig struct {
	DB *database.Queries
}

type requestHandler func(w http.ResponseWriter, r *http.Request, user database.User)

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	response := []byte(`{"error": "` + message + `"}`)
	_, _ = w.Write(response)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Failed to Marshal JSON Response %v ", err.Error()))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
}

func getApiKey(header http.Header) (string, error) {
	value := header.Get("Authorization")

	if value == "" {
		return "", errors.New("authorization header is missing")
	}
	values := strings.Split(value, " ")
	if len(values) != 2 {
		return "", errors.New("malformed authorization header")
	}
	if values[0] != "ApiKey" {
		return "", errors.New("authorization scheme must be ApiKey")
	}
	return values[1], nil
}

func (apiCfg *apiConfig) middlewareHandler(reqHandler requestHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey, err := getApiKey(r.Header)
		if err != nil {
			respondWithError(w, 401, err.Error())
		}
		user, err := apiCfg.DB.GetUserByApiKey(r.Context(), apiKey)
		if err != nil {
			respondWithError(w, 400, fmt.Sprintf("Couldn't get user %v ", err.Error()))
		}
		reqHandler(w, r, user)
	}
}

func (apiCfg *apiConfig) handleSignup(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Username string `json:"username"`
	}
	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error Parsing JSON: %v", err.Error()))
		return
	}
	user, err := apiCfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:       uuid.New(),
		Username: params.Username,
	})
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Couldn't create a user %v ", err.Error()))
		return
	}
	respondWithJSON(w, 201, user)

}

func (apiCfg *apiConfig) handleLogin(w http.ResponseWriter, r *http.Request, user database.User) {
	respondWithJSON(w, 200, user)
}

func (apiCfg *apiConfig) handleGetTimeRecords(w http.ResponseWriter, r *http.Request, user database.User) {
	timeRecords, err := apiCfg.DB.GetUserTimeRecords(r.Context(), user.ID)
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Couldn't get %v's time records  %v ", user.Username, err.Error()))
	}
	respondWithJSON(w, 200, timeRecords)
}

func (apiCfg *apiConfig) handleStartTimer(w http.ResponseWriter, r *http.Request, user database.User) {
	newTimeRecord, err := apiCfg.DB.CreateTimeRecord(r.Context(), database.CreateTimeRecordParams{
		ID:        uuid.New(),
		StartTime: time.Now().UTC(),
		UserID:    user.ID,
	})
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Couldn't start a timer %v ", err.Error()))
	}
	respondWithJSON(w, 201, newTimeRecord)
}

func (apiCfg *apiConfig) handleStopTimer(w http.ResponseWriter, r *http.Request) {
	timeRecordIdStr := chi.URLParam(r, "time_record_id")
	timeRecordID, err := uuid.Parse(timeRecordIdStr)
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Invalid time record ID %v ", err.Error()))
	}
	timeRecord, err := apiCfg.DB.UpdateTimeRecord(r.Context(), database.UpdateTimeRecordParams{
		ID: timeRecordID,
		StopTime: sql.NullTime{
			Time:  time.Now().UTC(),
			Valid: true,
		},
	})
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Couldn't stop a timer %v", err.Error()))
	}
	respondWithJSON(w, 200, timeRecord)
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	portString := os.Getenv("PORT")
	if portString == "" {
		log.Fatal("$PORT must be set")
	}

	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("$DATABASE_URL must be set")
	}
	sqlDB, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatal("Error connecting to the database ", err.Error())
	}
	apiConfig := apiConfig{DB: database.New(sqlDB)}

	v1router := chi.NewRouter()
	v1router.Post("/signup", apiConfig.handleSignup)
	v1router.Post("/login", apiConfig.middlewareHandler(apiConfig.handleLogin))
	v1router.Get("/user/time_records", apiConfig.middlewareHandler(apiConfig.handleGetTimeRecords))
	v1router.Post("/user/time_records/start", apiConfig.middlewareHandler(apiConfig.handleStartTimer))
	v1router.Post("/user/time_records/{time_record_id}/stop", apiConfig.handleStopTimer)

	router.Mount("/v1", v1router)
	server := &http.Server{
		Addr:    ":" + portString,
		Handler: router,
	}
	log.Println("Starting server on port: " + portString)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err.Error())
	}
}
