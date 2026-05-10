// Package slug converts free-form titles into URL/file-safe identifiers
// and handles collision suffixing (`-2`, `-3`, ...). Slugs are the only
// kind of ID exposed on the speechflow CLI and HTTP surface.
package slug

import (
	"fmt"
	"strings"
	"unicode"
)

// Checker reports whether a candidate slug is already taken. Implementations
// typically wrap a database lookup. Returning true means "this slug exists,
// pick another one."
type Checker func(candidate string) (bool, error)

// From converts s into a lowercase, hyphen-separated slug. Non-alphanumeric
// runes are coalesced into single hyphens; leading/trailing hyphens are
// trimmed. Returns "" for inputs that produce no alphanumeric runes.
func From(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	lastHyphen := true
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}

// Unique returns a slug derived from title that does not collide according
// to taken. The first candidate is the bare slug; subsequent collisions
// append "-2", "-3", and so on, mirroring the README's `pricing-strategy-2`
// example. Returns an error if title yields an empty slug or if taken
// returns an error.
func Unique(title string, taken Checker) (string, error) {
	base := From(title)
	if base == "" {
		return "", fmt.Errorf("slug: title %q produces an empty slug", title)
	}
	candidate := base
	for i := 2; ; i++ {
		exists, err := taken(candidate)
		if err != nil {
			return "", fmt.Errorf("slug: collision check: %w", err)
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i)
	}
}
