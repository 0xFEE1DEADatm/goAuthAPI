// @title Auth Service API
// @version 1.0
// @description API
// @host localhost:8080
// @BasePath /

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/0xFEE1DEADatm/goAuthAPI/internal/db"
	"github.com/0xFEE1DEADatm/goAuthAPI/internal/handler"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	database, err := db.ConnectDB()
	if err != nil {
		log.Fatal("DB connection error:", err)
	}
	defer database.Close()

	h := handler.NewHandler(database)

	r := mux.NewRouter()

	r.HandleFunc("/tokens", h.GetTokens).Methods("POST")             // получить пару токенов по userGUID
	r.HandleFunc("/tokens/refresh", h.RefreshTokens).Methods("POST") // обновить токены
	r.HandleFunc("/user", h.GetCurrentUser).Methods("GET")           // получить GUID текущего пользователя (защищённый роут)
	r.HandleFunc("/logout", h.Logout).Methods("POST")                // деавторизация

	// Swagger UI
	// r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server started at port", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
