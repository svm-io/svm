package display

import (
	"os"
	"testing"
)

func TestDetect(t *testing.T) {
	disp, err := Detect()
	if err != nil {
		t.Logf("Display detection: %v (expected in headless)", err)
		return
	}

	if disp.Type != "wayland" && disp.Type != "x11" {
		t.Errorf("unexpected display type: %s", disp.Type)
	}
}

func TestDisplayEnvWayland(t *testing.T) {
	disp := &Display{
		Type:   "wayland",
		Socket: "wayland-1",
		Secure: true,
	}

	env := disp.Env()
	if len(env) != 1 {
		t.Errorf("expected 1 env var, got %d", len(env))
	}

	if env[0] != "WAYLAND_DISPLAY=wayland-1" {
		t.Errorf("unexpected env: %s", env[0])
	}
}

func TestDisplayEnvX11(t *testing.T) {
	disp := &Display{
		Type:   "x11",
		Socket: ":0",
		Auth:   "/home/user/.Xauthority",
		Secure: false,
	}

	env := disp.Env()
	if len(env) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(env))
	}

	found := false
	for _, e := range env {
		if e == "DISPLAY=:0" {
			found = true
		}
	}
	if !found {
		t.Error("DISPLAY not in env")
	}
}

func TestDisplayIsSecureWayland(t *testing.T) {
	disp := &Display{Type: "wayland", Secure: true}
	if !disp.IsSecure() {
		t.Error("Wayland should be secure")
	}
}

func TestDisplayIsSecureX11(t *testing.T) {
	disp := &Display{Type: "x11", Secure: false}
	if disp.IsSecure() {
		t.Error("X11 should not be secure")
	}
}

func TestGetSocketPath(t *testing.T) {
	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime == "" {
		t.Skip("XDG_RUNTIME_DIR not set")
	}

	path, err := GetSocketPath("wayland-1")
	if err != nil {
		t.Errorf("GetSocketPath failed: %v", err)
	}

	if path == "" {
		t.Error("empty socket path")
	}
}

func TestGetAuthPath(t *testing.T) {
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	
	os.Setenv("HOME", "/root")
	_, err := GetAuthPath()
	if err != nil {
		t.Logf("GetAuthPath: %v", err)
	}
}

func TestDetectWaylandEnvVar(t *testing.T) {
	origSession := os.Getenv("XDG_SESSION_TYPE")
	origDisplay := os.Getenv("WAYLAND_DISPLAY")
	defer os.Setenv("XDG_SESSION_TYPE", origSession)
	defer os.Setenv("WAYLAND_DISPLAY", origDisplay)

	os.Setenv("XDG_SESSION_TYPE", "wayland")
	os.Setenv("WAYLAND_DISPLAY", "wayland-0")

	disp := detectWayland()
	if disp == nil {
		t.Error("should detect wayland from env")
	}
}

func TestDetectWaylandEnvVarSocket(t *testing.T) {
	origSession := os.Getenv("XDG_SESSION_TYPE")
	origDisplay := os.Getenv("WAYLAND_DISPLAY")
	defer os.Setenv("XDG_SESSION_TYPE", origSession)
	defer os.Setenv("WAYLAND_DISPLAY", origDisplay)

	os.Setenv("XDG_SESSION_TYPE", "wayland")
	os.Setenv("WAYLAND_DISPLAY", "wayland-test")

	disp := detectWayland()
	if disp == nil {
		t.Error("should detect wayland")
	}
	if disp != nil && disp.Socket != "wayland-test" {
		t.Errorf("wrong socket: %s", disp.Socket)
	}
}

func TestDetectX11Display(t *testing.T) {
	origDisplay := os.Getenv("DISPLAY")
	defer os.Setenv("DISPLAY", origDisplay)

	os.Setenv("DISPLAY", ":1")

	disp := detectX11()
	if disp == nil {
		t.Error("should detect x11")
	}
}

func TestDetectX11Default(t *testing.T) {
	origDisplay := os.Getenv("DISPLAY")
	defer os.Setenv("DISPLAY", origDisplay)

	os.Unsetenv("DISPLAY")

	disp := detectX11()
	if disp == nil {
		t.Error("should default to :0")
	}
	if disp != nil && disp.Socket != ":0" {
		t.Errorf("expected :0, got %s", disp.Socket)
	}
}

func TestDetectWaylandNoSocket(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)

	os.Unsetenv("XDG_RUNTIME_DIR")
	os.Unsetenv("XDG_SESSION_TYPE")
	os.Unsetenv("WAYLAND_DISPLAY")

	disp := detectWayland()
	if disp != nil {
		t.Logf("wayland detection: %v (fallback)", disp)
	}
}

func TestGetSocketPathEmpty(t *testing.T) {
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origXDG)

	os.Unsetenv("XDG_RUNTIME_DIR")

	_, err := GetSocketPath("test")
	if err == nil {
		t.Error("expected error when no XDG_RUNTIME_DIR")
	}
}