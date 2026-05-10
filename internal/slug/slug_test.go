package slug

import "testing"

func TestFrom(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Pricing Strategy", "pricing-strategy"},
		{"  Hello,  World!  ", "hello-world"},
		{"Q4 review", "q4-review"},
		{"---weird___input---", "weird-input"},
		{"already-a-slug", "already-a-slug"},
		{"123 Numbers", "123-numbers"},
		{"", ""},
		{"###", ""},
		{"Café", "café"},
	}
	for _, c := range cases {
		got := From(c.in)
		if got != c.want {
			t.Errorf("From(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestUnique_NoCollision(t *testing.T) {
	got, err := Unique("Pricing Strategy", func(string) (bool, error) { return false, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pricing-strategy" {
		t.Errorf("got %q, want %q", got, "pricing-strategy")
	}
}

func TestUnique_WithCollisions(t *testing.T) {
	taken := map[string]bool{
		"pricing-strategy":   true,
		"pricing-strategy-2": true,
		"pricing-strategy-3": true,
	}
	got, err := Unique("Pricing strategy", func(c string) (bool, error) {
		return taken[c], nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pricing-strategy-4" {
		t.Errorf("got %q, want %q", got, "pricing-strategy-4")
	}
}

func TestUnique_EmptyInput(t *testing.T) {
	_, err := Unique("!!!", func(string) (bool, error) { return false, nil })
	if err == nil {
		t.Fatalf("expected error for empty-slug input")
	}
}
