package lyrics

import (
	"regexp"
	"strings"
)

var artistSeparator = regexp.MustCompile(`(?i)\s*[,;&]\s*|\s+(?:feat\.?|ft\.?|featuring|x)\s+`)

func normalizedCredit(credit string) string {
	return strings.ToLower(strings.Join(strings.Fields(credit), " "))
}

// Uploaders may credit one performer while LRCLIB credits the collaboration.
// Compare complete credited names, accepting missing co-credits on either side
// but rejecting contradictory collaborator lists and partial artist names.
func artistCreditMatches(candidate, requested string) bool {
	candidate, requested = normalizedCredit(candidate), normalizedCredit(requested)
	if candidate == requested {
		return true
	}
	parts := func(credit string) map[string]bool {
		names := map[string]bool{}
		for _, name := range artistSeparator.Split(credit, -1) {
			if name = normalizedCredit(name); name != "" {
				names[name] = true
			}
		}
		return names
	}
	a, b := parts(candidate), parts(requested)
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	contains := func(all, subset map[string]bool) bool {
		for name := range subset {
			if !all[name] {
				return false
			}
		}
		return true
	}
	return contains(a, b) || contains(b, a)
}
