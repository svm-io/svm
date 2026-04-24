package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"vms/pkg/lxd"
)

var createCmd = &cobra.Command{
	Use:   "create <vm-name> [image]",
	Short: "Create a strict LXD VM",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		vmName := args[0]
		image := "ubuntu:noble"
		if len(args) > 1 {
			image = args[1]
		}

		client := lxd.New(ctx)

		if err := client.EnsureStrictProfile(); err != nil {
			return fmt.Errorf("failed to create strict profile: %w", err)
		}

		imageAlias, err := resolveImageAlias(ctx, image)
		if err != nil {
			return fmt.Errorf("failed to resolve image: %w", err)
		}

		if err := client.Launch(imageAlias, vmName, []string{"strict"}); err != nil {
			return fmt.Errorf("failed to launch VM: %w", err)
		}

		if err := client.WaitForRunning(vmName, 5*time.Minute); err != nil {
			client.Delete(vmName)
			return fmt.Errorf("VM did not start: %w", err)
		}

		if err := client.ApplySecurityRestrictions(vmName); err != nil {
			return fmt.Errorf("failed to apply restrictions: %w", err)
		}

		if err := client.WaitForDisplayAccess(vmName); err != nil {
			client.Delete(vmName)
			return fmt.Errorf("failed to setup display access: %w", err)
		}

		fmt.Printf("VM %q created with strict profile and display access\n", vmName)
		return nil
	},
}

func resolveImageAlias(ctx context.Context, input string) (string, error) {
	known := map[string]string{
		"ubuntu":       "ubuntu:noble",
		"ubuntu/24":    "ubuntu:noble",
		"noble":        "ubuntu:noble",
		"jammy":        "ubuntu:jammy",
		"24.04":        "ubuntu:noble",
		"22.04":        "ubuntu:jammy",
		"ubuntu/24.04": "ubuntu:noble",
	}
	if val, ok := known[input]; ok {
		return val, nil
	}
	if strings.Contains(input, ":") {
		return input, nil
	}
	listCmd := exec.CommandContext(ctx, lxcBinary, "image", "list", "images:"+input)
	out, err := listCmd.Output()
	if err != nil {
		return "", fmt.Errorf("image not found: %s", input)
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines[2:] {
		if strings.Contains(line, input) {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				return "images:" + parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("image not found in search results: %s", input)
}

func init() {
	rootCmd.AddCommand(createCmd)
}