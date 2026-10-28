package main

import (
	"context"
	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/banking"
	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/foundation"
	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/receipts"
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
	// Missing Receipt configuration disables only Receipt operations, never auth.
	receiptStore, _ := receipts.NewStore(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	receiptAPI := auth.ProtectMethods(receipts.NewAPI(receiptStore), "GET", "POST")
	mux.Handle("/api/v1/receipts", receiptAPI)
	mux.Handle("/api/v1/receipts/", receiptAPI)
	// Missing Banking configuration disables only Banking connections, never auth.
	if bankingAPI := newBankingAPI(); bankingAPI != nil {
		bankingAPI.Mount(mux, auth)
	}
	banking.MustRegisterProbe(mux, auth, os.Getenv) // VP-009c: removable
	server := &http.Server{Addr: ":" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	log.Fatal(server.ListenAndServe())
}

func newBankingAPI() *banking.API {
	ctx := context.Background()
	src, err := banking.NewSecretSource(ctx, os.Getenv)
	if err != nil {
		log.Print("Banking disabled: secret source not configured")
		return nil
	}
	keys, err := banking.LoadKeyring(ctx, src)
	if err != nil {
		log.Print("Banking disabled: token encryption key unavailable")
		return nil
	}
	env, err := banking.ParseEnv(os.Getenv("TELLER_ENV"))
	if err != nil {
		log.Print("Banking disabled: invalid TELLER_ENV")
		return nil
	}
	provider, err := banking.NewTellerClient(ctx, env, src)
	if err != nil {
		log.Print("Banking disabled: Teller client not configured")
		return nil
	}
	store, err := banking.NewSupabaseStore(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	if err != nil {
		log.Print("Banking disabled: store not configured")
		return nil
	}
	return banking.NewAPI(store, provider, keys)
}
