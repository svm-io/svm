package display

import (
	"fmt"
	"os"
	"path/filepath"
)

type Display struct {
	Type   string
	Socket string
	Auth   string
	Secure bool
}

func Detect() (*Display, error) {
	if wl := detectWayland(); wl != nil {
		return wl, nil
	}
	if x11 := detectX11(); x11 != nil {
		return x11, nil
	}
	return nil, fmt.Errorf("no display available on host")
}

func detectWayland() *Display {
	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime == "" {
		matches, _ := filepath.Glob("/run/user/*")
		if len(matches) > 0 {
			xdgRuntime = matches[0]
		}
	}
	if xdgRuntime == "" {
		return nil
	}
	matches, _ := filepath.Glob(filepath.Join(xdgRuntime, "wayland-*"))
	if len(matches) > 0 {
		return &Display{
			Type:   "wayland",
			Socket: filepath.Base(matches[0]),
			Secure: true,
		}
	}
	sessionType := os.Getenv("XDG_SESSION_TYPE")
	if sessionType == "wayland" {
		wlDisplay := os.Getenv("WAYLAND_DISPLAY")
		if wlDisplay != "" {
			return &Display{
				Type:   "wayland",
				Socket: wlDisplay,
				Secure: true,
			}
		}
	}
	return nil
}

func detectX11() *Display {
	display := os.Getenv("DISPLAY")
	if display == "" {
		return nil
	}
	xauthPath := os.Getenv("XAUTHORITY")
	if xauthPath == "" {
		home := os.Getenv("HOME")
		if home != "" {
			xauthPath = filepath.Join(home, ".Xauthority")
		}
	}
	return &Display{
		Type:   "x11",
		Socket: display,
		Auth:   xauthPath,
		Secure: false,
	}
}

func (d *Display) IsSecure() bool {
	return d.Secure
}

func (d *Display) Env() []string {
	switch d.Type {
	case "wayland":
		return []string{fmt.Sprintf("WAYLAND_DISPLAY=%s", d.Socket)}
	case "x11":
		return []string{
			fmt.Sprintf("DISPLAY=%s", d.Socket),
			fmt.Sprintf("XAUTHORITY=%s", d.Auth),
		}
	}
	return nil
}

func GetSocketPath(name string) (string, error) {
	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime == "" {
		matches, _ := filepath.Glob("/run/user/*")
		if len(matches) > 0 {
			xdgRuntime = matches[0]
		}
	}
	if xdgRuntime == "" {
		return "", fmt.Errorf("XDG_RUNTIME_DIR not set")
	}
	return filepath.Join(xdgRuntime, name), nil
}

func GetAuthPath() (string, error) {
	xauthPath := os.Getenv("XAUTHORITY")
	if xauthPath == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME not set and XAUTHORITY not found")
		}
		xauthPath = filepath.Join(home, ".Xauthority")
	}
	if _, err := os.Stat(xauthPath); err != nil {
		return "", fmt.Errorf(".Xauthority not found: %w", err)
	}
	return xauthPath, nil
}