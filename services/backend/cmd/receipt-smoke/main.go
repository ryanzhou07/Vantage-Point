// receipt-smoke is an explicitly opted-in, single paid-provider request using
// an embedded synthetic receipt. It never reads user receipt data or a database.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/ryanzhou07/Vantage-Point/services/backend/internal/receipts"
	"os"
	"time"
)

//go:embed testdata/synthetic.png
var synthetic []byte

func main() {
	if os.Getenv("RECEIPT_LIVE_SMOKE") != "synthetic-only" {
		fmt.Fprintln(os.Stderr, "No request sent. Opt in with RECEIPT_LIVE_SMOKE=synthetic-only; one Gemini request may consume quota.")
		os.Exit(2)
	}
	provider, err := receipts.NewGemini(os.Getenv("GEMINI_API_KEY"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "No request sent. GEMINI_API_KEY must be present in the private process environment.")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	proposal, err := provider.Extract(ctx, synthetic, "image/png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Gemini 3.5 Flash-Lite smoke failed: %v. Verify key/project/model access; no fallback was used.\n", err)
		os.Exit(1)
	}
	if proposal.Currency != "USD" || proposal.Subtotal != 1200 || proposal.Tax != 96 || proposal.Tip != 240 || proposal.Total != 1536 || len(proposal.Items) != 2 {
		fmt.Fprintln(os.Stderr, "Model access succeeded, but synthetic receipt extraction needs review. No financial result was accepted.")
		os.Exit(1)
	}
	fmt.Println("Gemini 3.5 Flash-Lite access and synthetic receipt extraction passed. No user receipt data was sent.")
}
