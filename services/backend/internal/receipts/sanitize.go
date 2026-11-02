package receipts

import "regexp"

// VP-028: card and payment details must never reach text other people can see.
var (
	cardRun    = regexp.MustCompile(`\d(?:[ -]?\d){12,18}`)
	cardMask   = regexp.MustCompile(`(?i)(?:\b(?:visa|mastercard|master card|amex|american express|discover)\b[ :#-]*)?(?:[*x•]{2,}[ -]?)+\d{2,4}\b|\b(?:ending|ends)(?: in)? *\d{4}\b`)
	authCode   = regexp.MustCompile(`(?i)\b(?:auth(?:orization)?|approval|appr)(?: code| no\.?)?[ :#]+[A-Z0-9]*\d[A-Z0-9]*\b`)
	onlyDigits = regexp.MustCompile(`\D`)
)

func luhn(d string) bool {
	sum, double := 0, false
	for i := len(d) - 1; i >= 0; i-- {
		n := int(d[i] - '0')
		if double {
			if n *= 2; n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return sum%10 == 0
}

// SanitizeText removes card numbers, masked card fragments and authorization codes from free text.
// It is for display text only; money fields never pass through it.
func SanitizeText(s string) string {
	s = cardRun.ReplaceAllStringFunc(s, func(m string) string {
		if d := onlyDigits.ReplaceAllString(m, ""); len(d) >= 13 && len(d) <= 19 && luhn(d) {
			return "[card removed]"
		}
		return m
	})
	s = cardMask.ReplaceAllString(s, "[card removed]")
	return authCode.ReplaceAllString(s, "[code removed]")
}
