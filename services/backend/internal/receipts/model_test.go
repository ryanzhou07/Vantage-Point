package receipts

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

func validDraft() Draft {
	return Draft{Proposal: Proposal{Currency: "USD", Items: []Item{{ID: "one", Name: "One", Quantity: 1, Amount: 100, ParticipantID: "a"}, {ID: "two", Name: "Two", Quantity: 2, Amount: 200, ParticipantID: "b"}}, Subtotal: 300, Tax: 5, Tip: 1, Total: 306}, Participants: []Participant{{ID: "b", Name: "Bee", Included: true}, {ID: "a", Name: "Aye", Included: true}}, TaxMethod: "equal", TipMethod: "proportional"}
}
func TestSplitConservesAndOrders(t *testing.T) {
	d := validDraft()
	p, e := Calculate(d)
	if e != nil {
		t.Fatal(e)
	}
	want := []Share{{"a", 100, 3, 0, 103}, {"b", 200, 2, 1, 203}}
	if !reflect.DeepEqual(p.Shares, want) || p.Total != 306 || !p.Draft {
		t.Fatalf("%+v", p)
	}
	d.Participants[0], d.Participants[1] = d.Participants[1], d.Participants[0]
	q, e := Calculate(d)
	if e != nil || !reflect.DeepEqual(p, q) {
		t.Fatal("input order affected remainder ties")
	}
}
func TestSplitProperties(t *testing.T) {
	rng := rand.New(rand.NewPCG(42, 7))
	for k := 0; k < 5000; k++ {
		d := validDraft()
		d.Items[0].Amount = int64(rng.IntN(100000))
		d.Items[1].Amount = int64(rng.IntN(100000)) + 1
		d.Subtotal = d.Items[0].Amount + d.Items[1].Amount
		d.Tax = int64(rng.IntN(10000))
		d.Tip = int64(rng.IntN(10000))
		d.Total = d.Subtotal + d.Tax + d.Tip
		if k%2 == 0 {
			d.TaxMethod = "proportional"
			d.TipMethod = "equal"
		}
		p, e := Calculate(d)
		if e != nil {
			t.Fatal(e)
		}
		var total, tax, tip int64
		for _, s := range p.Shares {
			if s.Items < 0 || s.Tax < 0 || s.Tip < 0 {
				t.Fatal("negative money")
			}
			total += s.Total
			tax += s.Tax
			tip += s.Tip
		}
		if total != d.Total || tax != d.Tax || tip != d.Tip {
			t.Fatal("money lost")
		}
	}
}
func TestSplitRejectsAmbiguityAndInvalidAssignments(t *testing.T) {
	cases := map[string]func(*Draft){"unassigned": func(d *Draft) { d.Items[0].ParticipantID = "" }, "unknown": func(d *Draft) { d.Items[0].ParticipantID = "other" }, "excluded": func(d *Draft) { d.Participants[1].Included = false }, "duplicate": func(d *Draft) { d.Participants[0].ID = "a" }, "reconcile": func(d *Draft) { d.Total++ }, "subtotal": func(d *Draft) { d.Subtotal++ }, "discount": func(d *Draft) { d.Discount = 1 }, "method": func(d *Draft) { d.TipMethod = "" }, "overflow": func(d *Draft) { d.Items[0].Amount = 1 << 62 }, "negative": func(d *Draft) { d.Tax = -1 }, "quantity": func(d *Draft) { d.Items[0].Quantity = 0 }, "zero weights": func(d *Draft) {
		for n := range d.Items {
			d.Items[n].Amount = 0
		}
		d.Subtotal = 0
		d.Total = d.Tax + d.Tip
	}, "blank participant": func(d *Draft) { d.Participants[0].Name = " " }}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := validDraft()
			mutate(&d)
			if _, e := Calculate(d); e == nil {
				t.Fatal("invalid split accepted")
			}
		})
	}
}
func TestZeroSubtotalEqualAndIncludedParticipants(t *testing.T) {
	d := validDraft()
	for n := range d.Items {
		d.Items[n].Amount = 0
	}
	d.Subtotal = 0
	d.Total = 6
	d.TipMethod = "equal"
	d.Participants = append(d.Participants, Participant{ID: "c", Name: "No share", Included: false})
	p, e := Calculate(d)
	if e != nil {
		t.Fatal(e)
	}
	if p.Shares[2].Total != 0 {
		t.Fatal("excluded participant charged")
	}
}
func TestLineAmountNotMultipliedByQuantity(t *testing.T) {
	d := validDraft()
	d.Items[1].Quantity = 1000
	p, e := Calculate(d)
	if e != nil || p.Shares[1].Items != 200 {
		t.Fatal("quantity treated as unit price")
	}
}
