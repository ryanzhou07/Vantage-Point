package main

import (
	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/foundation"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	auth, err := foundation.New(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_ANON_KEY"), os.Getenv("WEB_ORIGIN"))
	if err != nil {
		log.Fatal("Authentication configuration is required")
	}
	mux.Handle("/api/v1/me", auth.Protect(http.HandlerFunc(auth.Me)))
	server := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	log.Fatal(server.ListenAndServe())
}
