package cmd

import (
	"context"
	"testing"
	"time"
)

func TestResolveImageAliasKnown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ubuntu", "ubuntu:noble"},
		{"noble", "ubuntu:noble"},
		{"jammy", "ubuntu:jammy"},
		{"24.04", "ubuntu:noble"},
		{"ubuntu:noble", "ubuntu:noble"},
		{"images:ubuntu/noble", "images:ubuntu/noble"},
	}

	for _, tc := range tests {
		ctx := context.Background()
		alias, err := resolveImageAlias(ctx, tc.input)
		if err != nil {
			t.Errorf("resolveImageAlias(%s): %v", tc.input, err)
			continue
		}
		if alias != tc.expected {
			t.Errorf("resolveImageAlias(%s) = %s, want %s", tc.input, alias, tc.expected)
		}
	}
}

func TestResolveImageAliasUnknown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	alias, err := resolveImageAlias(ctx, "nonexistent-image-xyz")
	if err != nil {
		t.Logf("expected fallback: %v", err)
	}

	if alias == "" {
		t.Error("expected fallback to default")
	}
}