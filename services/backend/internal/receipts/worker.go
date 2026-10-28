package receipts

import (
	"context"
	"errors"
	"time"
)

type Job struct {
	Receipt  Record `json:"receipt"`
	Lease    string `json:"lease_token"`
	Attempts int    `json:"attempts"`
}
type Worker struct {
	Store    *Store
	Provider Extractor
}

// Run processes one job at a time. Multiple worker processes remain fenced by leases.
// Polling never starts work in API handlers. A crash can repeat one in-flight paid
// request after lease expiry; provider-side exactly-once cannot be promised.
func (w *Worker) Run(ctx context.Context) error {
	if w.Store == nil || w.Provider == nil {
		return errors.New("Receipt worker configuration is required")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		err := w.Step(ctx)
		delay := time.Second
		if err != nil {
			delay = 5 * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}
func (w *Worker) Step(ctx context.Context) error {
	var job *Job
	if e := w.Store.rpc(ctx, "receipt_claim", map[string]any{}, &job); e != nil {
		return e
	}
	if job == nil {
		return nil
	}
	work, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	data, e := w.Store.image(work, job.Receipt)
	var p Proposal
	if e == nil {
		p, e = w.Provider.Extract(work, data, job.Receipt.ImageMIME)
		if e == nil && ValidateProposal(p) != nil {
			e = &ProviderError{Code: "VISION_INVALID_OUTPUT", Retryable: false}
		}
	}
	code := ""
	retry := false
	if e != nil {
		code = "VISION_UNAVAILABLE"
		retry = true
		var pe *ProviderError
		if errors.As(e, &pe) {
			code = pe.Code
			retry = pe.Retryable
		}
	}
	var proposal any
	var failure any
	if e == nil {
		proposal = p
	} else {
		failure = code
	}
	// Completion has an independent short budget; cancelled provider work must still
	// release/fail its lease when possible. If shutdown prevents it, lease recovery wins.
	finish, cancelFinish := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelFinish()
	var accepted bool
	return w.Store.rpc(finish, "receipt_finish", map[string]any{"p_id": job.Receipt.ID, "p_lease": job.Lease, "p_proposal": proposal, "p_error": failure, "p_retry": retry}, &accepted)
}
