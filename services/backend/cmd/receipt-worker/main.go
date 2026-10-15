package main

import (
	"context"
	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/receipts"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	store, err := receipts.NewStore(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_ROLE_KEY"))
	if err != nil {
		log.Fatal("Receipt worker requires server-side Supabase configuration")
	}
	var provider receipts.Extractor
	switch os.Getenv("RECEIPT_VISION_PROVIDER") {
	case "gemini":
		provider, err = receipts.NewGemini(os.Getenv("GEMINI_API_KEY"))
	case "fake":
		if os.Getenv("RECEIPT_ALLOW_FAKE") != "development-test-only" {
			log.Fatal("Fake vision requires explicit development-test-only opt-in")
		}
		provider = receipts.FakeVision{}
	default:
		log.Fatal("Set RECEIPT_VISION_PROVIDER=gemini; no automatic provider or paid fallback")
	}
	if err != nil {
		log.Fatal("Gemini key/access configuration is required; see Receipt setup documentation")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	worker := receipts.Worker{Store: store, Provider: provider}
	if err = worker.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal("Receipt worker stopped")
	}
}
