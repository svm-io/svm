package lxd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const lxcBinary = "/snap/bin/lxc"

type Client struct {
	ctx context.Context
}

func New(ctx context.Context) *Client {
	return &Client{ctx: ctx}
}

func (c *Client) Launch(image, name string, profiles []string) error {
	args := []string{"launch", image, name}
	for _, p := range profiles {
		args = append(args, "-p", p)
	}
	cmd := exec.CommandContext(c.ctx, lxcBinary, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("lxc launch failed: %w", err)
	}
	return nil
}

func (c *Client) Start(name string) error {
	cmd := exec.CommandContext(c.ctx, lxcBinary, "start", name)
	return cmd.Run()
}

func (c *Client) Stop(name string) error {
	cmd := exec.CommandContext(c.ctx, lxcBinary, "stop", name)
	return cmd.Run()
}

func (c *Client) Delete(name string) error {
	exec.CommandContext(c.ctx, lxcBinary, "stop", name, "-f").Run()
	time.Sleep(2 * time.Second)
	cmd := exec.CommandContext(c.ctx, lxcBinary, "delete", name, "--force")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	return nil
}

func (c *Client) Info(name string) (map[string]string, error) {
	cmd := exec.CommandContext(c.ctx, lxcBinary, "info", name)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	info := make(map[string]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			info[key] = value
		}
	}
	return info, nil
}

func (c *Client) State(name string) (string, error) {
	info, err := c.Info(name)
	if err != nil {
		return "", err
	}
	return info["Status"], nil
}

func (c *Client) WaitForRunning(name string, timeout time.Duration) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	deadline := time.After(timeout)
	
	for {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		case <-deadline:
			return fmt.Errorf("timeout waiting for container %s to be running", name)
		case <-ticker.C:
			cmd := exec.CommandContext(c.ctx, lxcBinary, "exec", name, "--", "true")
			err := cmd.Run()
			if err == nil {
				return nil
			}
		}
	}
}

func (c *Client) ConfigGet(name, key string) (string, error) {
	cmd := exec.CommandContext(c.ctx, lxcBinary, "config", "get", name, key)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("config get %s failed: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) ConfigSet(name, key, value string) error {
	cmd := exec.CommandContext(c.ctx, lxcBinary, "config", "set", name, key, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("config set %s=%s failed: %w", key, value, err)
	}
	return nil
}

func (c *Client) ApplySecurityRestrictions(name string) error {
	restrictions := map[string]string{
		"security.nesting":     "false",
		"security.privileged":  "false",
		"linux.kernel_modules": "",
	}
	for key, value := range restrictions {
		if err := c.ConfigSet(name, key, value); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Exec(name string, env []string, command ...string) error {
	args := []string{"exec", name, "--"}
	if len(env) > 0 {
		args = append(args, "env")
		args = append(args, env...)
	}
	args = append(args, command...)
	cmd := exec.CommandContext(c.ctx, lxcBinary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (c *Client) FilePush(localPath, remotePath string) error {
	cmd := exec.CommandContext(c.ctx, lxcBinary, "file", "push", localPath, remotePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("file push failed: %w", err)
	}
	return nil
}

func (c *Client) ExecToString(name string, command ...string) (string, error) {
	args := []string{"exec", name, "--"}
	args = append(args, command...)
	cmd := exec.CommandContext(c.ctx, lxcBinary, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (c *Client) ProfileList() ([]string, error) {
	cmd := exec.CommandContext(c.ctx, lxcBinary, "profile", "list", "--format", "csv")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var profiles []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "name") {
			parts := strings.Split(line, ",")
			if len(parts) > 0 {
				profiles = append(profiles, strings.TrimSpace(parts[0]))
			}
		}
	}
	return profiles, nil
}

func (c *Client) ProfileCreate(name string, config map[string]string) error {
	cmd := exec.CommandContext(c.ctx, lxcBinary, "profile", "create", name)
	if err := cmd.Run(); err != nil {
		return err
	}
	for key, value := range config {
		cmd := exec.CommandContext(c.ctx, lxcBinary, "profile", "set", name, key, value)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ProfileExists(name string) bool {
	profiles, err := c.ProfileList()
	if err != nil {
		return false
	}
	for _, p := range profiles {
		if p == name {
			return true
		}
	}
	return false
}

func (c *Client) EnsureStrictProfile() error {
	if c.ProfileExists("strict") {
		return nil
	}
	config := map[string]string{
		"security.nesting":     "false",
		"security.privileged":  "false",
		"security.protocols":   "clear",
		"linux.kernel_modules": "",
		"limits.cpu":           "2",
		"limits.memory":        "2GiB",
	}
	return c.ProfileCreate("strict", config)
}