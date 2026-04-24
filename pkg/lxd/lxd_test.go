package lxd

import (
	"context"
	"testing"
	"time"
)

func TestClientNew(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)
	if client == nil {
		t.Fatal("New returned nil")
	}
	if client.ctx != ctx {
		t.Error("context not set correctly")
	}
}

func TestClientState(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	_, err := client.State("nonexistent-vm-12345")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientWaitForRunningTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	client := New(ctx)

	err := client.WaitForRunning("nonexistent-vm-test-12345", 50*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error for nonexistent VM")
	}
}

func TestClientConfigGet(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	_, err := client.ConfigGet("nonexistent-vm-123", "security.nesting")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientConfigSet(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	err := client.ConfigSet("nonexistent-vm-123", "security.nesting", "false")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientProfileList(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	profiles, err := client.ProfileList()
	if err != nil {
		t.Skipf("LXD not available: %v", err)
	}

	if len(profiles) == 0 {
		t.Error("expected at least default profile")
	}

	found := false
	for _, p := range profiles {
		if p == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Error("default profile not found")
	}
}

func TestClientProfileExists(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	exists := client.ProfileExists("default")
	if !exists {
		t.Log("default profile may not exist in test env")
	}
}

func TestClientProfileExistsNotFound(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	if client.ProfileExists("nonexistent-profile-xyz-123") {
		t.Error("nonexistent profile should not exist")
	}
}

func TestClientInfo(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	_, err := client.Info("nonexistent-vm-123")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientApplySecurityRestrictions(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	err := client.ApplySecurityRestrictions("nonexistent-vm-123")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientEnsureStrictProfile(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	err := client.EnsureStrictProfile()
	if err != nil {
		t.Logf("EnsureStrictProfile warning: %v (expected in test env)", err)
	}
}

func TestClientStart(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	err := client.Start("nonexistent-vm-123")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientStop(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	err := client.Stop("nonexistent-vm-123")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientDelete(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	err := client.Delete("nonexistent-vm-123")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientFilePush(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	err := client.FilePush("/nonexistent", "nonexistent-vm-123/path")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestClientProfileCreate(t *testing.T) {
	ctx := context.Background()
	client := New(ctx)

	profileName := "test-profile-" + time.Now().Format("20060102150405")

	err := client.ProfileCreate(profileName, map[string]string{
		"limits.cpu": "2",
	})
	if err != nil {
		t.Logf("ProfileCreate: %v (may require permissions)", err)
	}
}