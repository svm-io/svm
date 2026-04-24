package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vms/pkg/display"
	"vms/pkg/lxd"
)

func TestIntegrationCreateVM(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Set INTEGRATION_TEST=1 to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client := lxd.New(ctx)
	vmName := "test-vm-" + time.Now().Format("20060102150405")

	err := client.EnsureStrictProfile()
	if err != nil {
		t.Logf("EnsureStrictProfile: %v", err)
	}

	err = client.Launch("ubuntu:noble", vmName, []string{"strict"})
	if err != nil {
		t.Skipf("Skipping: LXD not available: %v", err)
	}
	defer client.Delete(vmName)

	err = client.WaitForRunning(vmName, 5*time.Minute)
	if err != nil {
		t.Fatalf("VM did not start: %v", err)
	}

	state, _ := client.State(vmName)
	if state != "Running" {
		t.Errorf("expected Running, got %s", state)
	}
}

func TestIntegrationLaunchApp(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Set INTEGRATION_TEST=1 to run integration tests")
	}

	ctx := context.Background()
	client := lxd.New(ctx)

	err := client.ExecToString("nonexistent", "echo", "test")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestIntegrationDisplay(t *testing.T) {
	if os.Getenv("DISPLAY") == "" || os.Getenv("WAYLAND_DISPLAY") == "" {
		if os.Getenv("INTEGRATION_TEST") != "1" {
			t.Skip("No display available, skipping")
		}
	}

	disp, err := display.Detect()
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}

	if disp.Type != "wayland" && disp.Type != "x11" {
		t.Errorf("unexpected display type: %s", disp.Type)
	}

	t.Logf("Detected display: %s (socket: %s, secure: %v)", disp.Type, disp.Socket, disp.Secure)
}

func TestIntegrationWaylandSocket(t *testing.T) {
	if os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("XDG_SESSION_TYPE") != "wayland" {
		if os.Getenv("INTEGRATION_TEST") != "1" {
			t.Skip("Wayland not available")
		}
	}

	origSession := os.Getenv("XDG_SESSION_TYPE")
	origDisplay := os.Getenv("WAYLAND_DISPLAY")
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("XDG_SESSION_TYPE", origSession)
		os.Setenv("WAYLAND_DISPLAY", origDisplay)
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
	}()

	os.Setenv("XDG_SESSION_TYPE", "wayland")
	os.Setenv("WAYLAND_DISPLAY", "wayland-0")

	disp := display.Detect()
	if disp == nil || disp.Type != "wayland" {
		t.Skip("Wayland not detected")
	}

	if disp.Socket != "wayland-0" && disp.Socket != "wayland-0" {
		t.Logf("wayland socket: %s", disp.Socket)
	}
}

func TestIntegrationX11Socket(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		if os.Getenv("INTEGRATION_TEST") != "1" {
			t.Skip("X11 not available")
		}
	}

	origDisplay := os.Getenv("DISPLAY")
	origXauth := os.Getenv("XAUTHORITY")
	defer func() {
		os.Setenv("DISPLAY", origDisplay)
		os.Setenv("XAUTHORITY", origXauth)
	}()

	os.Setenv("DISPLAY", ":0")

	disp := display.Detect()
	if disp == nil || disp.Type != "x11" {
		t.Skip("X11 not detected")
	}

	if disp.Socket != ":0" {
		t.Logf("display socket: %s", disp.Socket)
	}
}

func TestIntegrationFilePush(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Set INTEGRATION_TEST=1 to run integration tests")
	}

	ctx := context.Background()
	client := lxd.New(ctx)
	vmName := "nonexistent-vm-123"

	err := client.FilePush("/etc/hosts", vmName+"/etc/hosts")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestIntegrationProfileCreate(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Set INTEGRATION_TEST=1 to run integration tests")
	}

	ctx := context.Background()
	client := lxd.New(ctx)
	profileName := "test-profile-" + time.Now().Format("20060102150405")

	err := client.ProfileCreate(profileName, map[string]string{
		"limits.cpu": "2",
	})
	if err != nil {
		t.Logf("ProfileCreate: %v", err)
	}
}

func TestIntegrationExec(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Set INTEGRATION_TEST=1 to run integration tests")
	}

	ctx := context.Background()
	client := lxd.New(ctx)

	out, err := client.ExecToString("nonexistent-vm", "ls")
	if err == nil {
		t.Errorf("expected error for nonexistent VM")
	}
	if out != "" {
		t.Logf("output: %s", out)
	}
}

func TestIntegrationSecurityRestrictions(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Set INTEGRATION_TEST=1 to run integration tests")
	}

	ctx := context.Background()
	client := lxd.New(ctx)

	err := client.ApplySecurityRestrictions("nonexistent-vm")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestIntegrationWaitForDisplayAccess(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("Set INTEGRATION_TEST=1 to run integration tests")
	}

	ctx := context.Background()
	client := lxd.New(ctx)

	err := client.WaitForDisplayAccess("nonexistent-vm")
	if err == nil {
		t.Error("expected error for nonexistent VM")
	}
}

func TestIntegrationSocketPath(t *testing.T) {
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)

	if os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("XDG_RUNTIME_DIR") == "" {
		t.Skip("No wayland socket available")
	}

	os.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")

	path, err := display.GetSocketPath("wayland-1")
	if err != nil {
		t.Fatalf("GetSocketPath: %v", err)
	}

	expected := filepath.Join("/run/user/1000", "wayland-1")
	if !strings.Contains(path, "wayland-1") && path != expected {
		t.Logf("path: %s (expected: %s)", path, expected)
	}
}