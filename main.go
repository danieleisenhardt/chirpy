package main

import (
	"database/sql"
	"os"

	"github.com/danieleisenhardt/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbConfig       *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	_ = godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)

	port := "8080"

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	apiCfg := &apiConfig{
		atomic.Int32{},
		dbQueries,
	}

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type parameters = struct {
			Body   string        `json:"body"`
			UserID uuid.NullUUID `json:"user_id"`
		}
		type successResponse = struct {
			ID        uuid.UUID     `json:"id"`
			CreatedAt string        `json:"created_at"`
			UpdatedAt string        `json:"updated_at"`
			Body      string        `json:"body"`
			UserID    uuid.NullUUID `json:"user_id"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			log.Printf("Error decoding JSON: %s", err)
			respondWithError(w, http.StatusBadRequest, "Error decoding JSON")
			return
		}
		if len(params.Body) > 140 {
			respondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}
		cleanedBody := replaceBadWords(params.Body)

		//chirp, err := apiCfg.dbConfig.CreateChirp(r.Context(), cleanedBody, params.UserID)
		chirp, err := apiCfg.dbConfig.CreateChirp(r.Context(), database.CreateChirpParams{Body: cleanedBody, UserID: params.UserID})
		if err != nil {
			log.Printf("Error saving user: %s", err)
			respondWithError(w, http.StatusInternalServerError, "Error saving user")
			return
		}
		respondWithJSON(w, http.StatusCreated, successResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters = struct {
			Email string `json:"email"`
		}
		type successResponse = struct {
			Id        uuid.UUID `json:"id"`
			CreatedAt string    `json:"created_at"`
			UpdatedAt string    `json:"updated_at"`
			Email     string    `json:"email"`
		}

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			log.Printf("Error decoding JSON: %s", err)
			respondWithError(w, http.StatusBadRequest, "Error decoding JSON")
			return
		}

		user, err := apiCfg.dbConfig.CreateUser(r.Context(), params.Email)
		if err != nil {
			log.Printf("Error saving user: %s", err)
			respondWithError(w, http.StatusInternalServerError, "Error saving user")
			return
		}

		respondWithJSON(w, http.StatusCreated, successResponse{
			Id:        user.ID,
			CreatedAt: user.CreatedAt.String(),
			UpdatedAt: user.UpdatedAt.String(),
			Email:     user.Email,
		})
	})

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		body := "<html>\n<body>\n<h1>Welcome, Chirpy Admin</h1>\n<p>Chirpy has been visited %d times!</p>\n</body>\n</html>"
		_, _ = w.Write([]byte(fmt.Sprintf(body, apiCfg.fileserverHits.Load())))
	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if platform != "dev" {
			respondWithError(w, http.StatusBadRequest, "Endpoint not available on this platform")
			return
		}

		err := apiCfg.dbConfig.TruncateUsers(r.Context())
		if err != nil {
			log.Printf("Error truncating users table")
			respondWithError(w, http.StatusInternalServerError, "Error truncating users table")
			return
		}
		apiCfg.fileserverHits.Store(0)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	handler := http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
