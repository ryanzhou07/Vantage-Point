// Package receipts owns receipt drafts; model proposals never establish verified truth.
package receipts

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const MaxMoney int64 = 100000000 // $1m; keeps bounded multiplication exact in int64.
const MaxItems = 200
const MaxParticipants = 40

type Item struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Quantity       int      `json:"quantity"`
	Amount         int64    `json:"amount_minor"`    // Complete line amount, never unit price.
	ParticipantIDs []string `json:"participant_ids"` // Claimants share the line evenly.
}
type Participant struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Included bool   `json:"included"`
	IsOwner  bool   `json:"is_owner"`
}
type Proposal struct {
	Merchant string `json:"merchant"`
	Date     string `json:"date"`
	Currency string `json:"currency"`
	Items    []Item `json:"items"`
	Subtotal int64  `json:"subtotal_minor"`
	Tax      int64  `json:"tax_minor"`
	Tip      int64  `json:"tip_minor"`
	Discount int64  `json:"discount_minor"`
	Total    int64  `json:"total_minor"`
}
type Draft struct {
	Proposal
	Participants []Participant `json:"participants"`
	TaxMethod    string        `json:"tax_method"`
	TipMethod    string        `json:"tip_method"`
}
type Share struct {
	ParticipantID string `json:"participant_id"`
	Items         int64  `json:"items_minor"`
	Tax           int64  `json:"tax_minor"`
	Tip           int64  `json:"tip_minor"`
	Total         int64  `json:"total_minor"`
}
type Preview struct {
	Shares []Share `json:"shares"`
	Total  int64   `json:"total_minor"`
	Draft  bool    `json:"draft"`
}

func bounded(s string, max int) bool {
	return strings.TrimSpace(s) != "" && len(s) <= max && !strings.ContainsRune(s, '\x00')
}
func ValidateProposal(p Proposal) error {
	if len(p.Merchant) > 200 || len(p.Date) > 32 || p.Currency != "USD" || len(p.Items) == 0 || len(p.Items) > MaxItems {
		return errors.New("Check merchant, currency (USD), and 1–200 item lines.")
	}
	for _, n := range []int64{p.Subtotal, p.Tax, p.Tip, p.Discount, p.Total} {
		if n < 0 || n > MaxMoney {
			return errors.New("Amounts must be nonnegative integer cents within the supported limit.")
		}
	}
	seen := map[string]bool{}
	var sum int64
	for _, i := range p.Items {
		if !bounded(i.ID, 64) || seen[i.ID] || !bounded(i.Name, 200) || i.Quantity < 1 || i.Quantity > 1000 || i.Amount < 0 || i.Amount > MaxMoney {
			return errors.New("Check item IDs, names, quantities and complete line amounts.")
		}
		seen[i.ID] = true
		sum += i.Amount
	}
	if sum > MaxMoney {
		return errors.New("Item sum exceeds the supported limit.")
	}
	return nil
}
func Calculate(d Draft) (Preview, error) {
	fail := func(msg string) (Preview, error) { return Preview{}, errors.New(msg) }
	if err := ValidateProposal(d.Proposal); err != nil {
		return Preview{}, err
	}
	if d.Discount != 0 {
		return fail("Receipt-level discounts need owner resolution in item line amounts before a verified preview. Preserve the original image/proposal.")
	}
	if len(d.Participants) == 0 || len(d.Participants) > MaxParticipants {
		return fail("Add 1–40 participants.")
	}
	people := append([]Participant(nil), d.Participants...)
	sort.Slice(people, func(i, j int) bool { return people[i].ID < people[j].ID })
	byID := map[string]int{}
	shares := make([]Share, len(people))
	for n, p := range people {
		if !bounded(p.ID, 64) || !bounded(p.Name, 100) {
			return fail("Check participant names and IDs.")
		}
		if _, ok := byID[p.ID]; ok {
			return fail("Participant IDs must be unique.")
		}
		byID[p.ID] = n
		shares[n].ParticipantID = p.ID
	}
	var sum int64
	for _, item := range d.Items {
		if len(item.ParticipantIDs) == 0 {
			return fail("Assign every item line to at least one participant.")
		}
		idx := make([]int, 0, len(item.ParticipantIDs))
		seen := map[int]bool{}
		for _, id := range item.ParticipantIDs {
			n, ok := byID[id]
			if !ok || seen[n] {
				return fail("Assign every item line to existing, distinct participants.")
			}
			if !people[n].Included {
				return fail("Participants with assigned items must be included.")
			}
			seen[n] = true
			idx = append(idx, n)
		}
		sort.Ints(idx) // people is ID-sorted, so the remainder cents land deterministically.
		ones := make([]int64, len(idx))
		for k := range ones {
			ones[k] = 1
		}
		for k, v := range distribute(item.Amount, ones, int64(len(idx))) {
			shares[idx[k]].Items += v
		}
		sum += item.Amount
	}
	if sum != d.Subtotal || sum+d.Tax+d.Tip != d.Total {
		return fail(fmt.Sprintf("Reconcile item subtotal and receipt total before calculating (items: %d cents).", sum))
	}
	for _, part := range []struct {
		amount int64
		method string
		tax    bool
	}{{d.Tax, d.TaxMethod, true}, {d.Tip, d.TipMethod, false}} {
		weights := make([]int64, len(people))
		var weight int64
		if part.method != "equal" && part.method != "proportional" {
			return fail("Choose equal or proportional separately for tax and tip.")
		}
		for n, p := range people {
			if p.Included {
				if part.method == "equal" {
					weights[n] = 1
				} else {
					weights[n] = shares[n].Items
				}
				weight += weights[n]
			}
		}
		if weight == 0 {
			return fail("Include participants with a positive subtotal for proportional allocation, or choose equal with included participants.")
		}
		// Everyone gets the floor of their share; the leftover cents land on the owner
		// (who fronts the bill) so no guest pays a rounded-up cent. Falls back to the first included person.
		values := make([]int64, len(weights))
		var used int64
		for n, w := range weights {
			values[n] = part.amount * w / weight
			used += values[n]
		}
		rest := -1
		for n, p := range people {
			if p.Included && (rest < 0 || p.IsOwner) {
				rest = n
			}
			if p.Included && p.IsOwner {
				break
			}
		}
		values[rest] += part.amount - used
		for n, v := range values {
			if part.tax {
				shares[n].Tax = v
			} else {
				shares[n].Tip = v
			}
		}
	}
	for n := range shares {
		shares[n].Total = shares[n].Items + shares[n].Tax + shares[n].Tip
	}
	return Preview{Shares: shares, Total: d.Total, Draft: true}, nil
}

// Largest remainder; stable participant-ID order breaks equal remainder ties.
func distribute(amount int64, weights []int64, total int64) []int64 {
	out := make([]int64, len(weights))
	order := make([]int, len(weights))
	var used int64
	for n, w := range weights {
		out[n] = amount * w / total
		used += out[n]
		order[n] = n
	}
	sort.SliceStable(order, func(i, j int) bool { return amount*weights[order[i]]%total > amount*weights[order[j]]%total })
	for _, n := range order {
		if used == amount {
			break
		}
		if weights[n] > 0 {
			out[n]++
			used++
		}
	}
	return out
}

// canonicalDraftID converts legacy advisory/client keys to receipt-scoped UUIDs.
// MD5 is only a stable identifier mapping, never a credential or auth mechanism.
// PostgreSQL receipt_internal_id uses the same byte string and UUID formatting.
func canonicalDraftID(receipt, kind, value string) string {
	if receiptID.MatchString(strings.ToLower(value)) {
		return strings.ToLower(value)
	}
	sum := md5.Sum([]byte(receipt + ":" + kind + ":" + value))
	raw := hex.EncodeToString(sum[:])
	return raw[:8] + "-" + raw[8:12] + "-" + raw[12:16] + "-" + raw[16:20] + "-" + raw[20:]
}
func canonicalizeDraft(receipt string, d Draft) Draft {
	for n := range d.Participants {
		d.Participants[n].ID = canonicalDraftID(receipt, "participant", d.Participants[n].ID)
	}
	for n := range d.Items {
		d.Items[n].ID = canonicalDraftID(receipt, "item", d.Items[n].ID)
		ids := []string{}
		for _, id := range d.Items[n].ParticipantIDs {
			ids = append(ids, canonicalDraftID(receipt, "participant", id))
		}
		d.Items[n].ParticipantIDs = ids
	}
	return d
}
