// @title Auth Service API
// @version 1.0
// @description API
// @host localhost:8080
// @BasePath /

package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	_ "github.com/0xFEE1DEADatm/goAuthAPI/docs"
	"github.com/0xFEE1DEADatm/goAuthAPI/internal/db"
	"github.com/0xFEE1DEADatm/goAuthAPI/internal/handler"
	"github.com/0xFEE1DEADatm/goAuthAPI/internal/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func ensureAuthSessionsTable(database *sql.DB) error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS auth_sessions (
		user_guid UUID PRIMARY KEY,
		refresh_token TEXT NOT NULL,
		user_agent TEXT,
		ip TEXT,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);
	`
	_, err := database.Exec(createTableQuery)
	return err
}
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	database, err := db.ConnectDB()
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	defer database.Close()

	if err := ensureAuthSessionsTable(database); err != nil {
		log.Fatalf("Failed to create auth_sessions table: %v", err)
	}

	h := handler.NewHandler(database)

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK - Root is working"))
	}).Methods("GET")

	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	r.HandleFunc("/tokens", h.GetTokens).Methods("POST")
	r.HandleFunc("/tokens/refresh", h.RefreshTokens).Methods("POST")

	r.Handle("/me", middleware.AuthMiddleware(http.HandlerFunc(h.GetCurrentUser))).Methods("GET")
	r.Handle("/logout", middleware.AuthMiddleware(http.HandlerFunc(h.Logout))).Methods("POST")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server started at port", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
