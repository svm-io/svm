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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize LXD for VMS (one-time setup)",
	Long: `Configure LXD with required storage pool, network, and default profile.
This command must be run once before creating VMs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		return runInit(ctx)
	},
}

func runInit(ctx context.Context) error {
	fmt.Println("Initializing LXD for VMS...")
	fmt.Println()

	fmt.Print("[1/3] Checking storage pool... ")
	if err := ensureStoragePool(ctx); err != nil {
		return fmt.Errorf("storage pool setup failed: %w", err)
	}
	fmt.Println("OK")

	fmt.Print("[2/3] Checking network... ")
	if err := ensureNetwork(ctx); err != nil {
		return fmt.Errorf("network setup failed: %w", err)
	}
	fmt.Println("OK")

	fmt.Print("[3/3] Configuring default profile... ")
	if err := configureDefaultProfile(ctx); err != nil {
		return fmt.Errorf("profile configuration failed: %w", err)
	}
	fmt.Println("OK")

	fmt.Println()
	fmt.Println("✓ LXD is ready for VMS")
	fmt.Println()
	fmt.Println("Next step:")
	fmt.Println("  vms host-setup        # Configure host for display")
	fmt.Println("  vms create myvm       # Create your first VM")
	fmt.Println()

	return nil
}

func ensureStoragePool(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "storage", "show", "default")
	if err := cmd.Run(); err == nil {
		return nil
	}

	cmd = exec.CommandContext(ctx, lxcBinary, "storage", "create", "default", "dir")
	return cmd.Run()
}

func ensureNetwork(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "network", "show", "lxdbr0")
	if err := cmd.Run(); err == nil {
		return nil
	}

	cmd = exec.CommandContext(ctx, lxcBinary, "network", "create", "lxdbr0")
	return cmd.Run()
}

func configureDefaultProfile(ctx context.Context) error {
	client := lxd.New(ctx)

	profileCmd := exec.CommandContext(ctx, lxcBinary, "profile", "show", "default")
	profileOutput, err := profileCmd.Output()
	if err != nil {
		return fmt.Errorf("cannot read default profile: %w", err)
	}

	profileStr := string(profileOutput)

	if !strings.Contains(profileStr, "root:") || !strings.Contains(profileStr, "pool: default") {
		diskCmd := exec.CommandContext(ctx, lxcBinary, "profile", "device", "add", "default", "root", "disk", "path=/", "pool=default")
		if err := diskCmd.Run(); err != nil {
			return fmt.Errorf("cannot add root disk: %w", err)
		}
	}

	if !strings.Contains(profileStr, "eth0:") || !strings.Contains(profileStr, "lxdbr0") {
		netCmd := exec.CommandContext(ctx, lxcBinary, "profile", "device", "add", "default", "eth0", "nic", "network=lxdbr0", "name=eth0")
		if err := netCmd.Run(); err != nil {
			return fmt.Errorf("cannot add network: %w", err)
		}
	}

	if !client.ProfileExists("strict") {
		config := map[string]string{
			"security.nesting":     "false",
			"security.privileged":  "false",
			"security.protocols":   "clear",
			"linux.kernel_modules": "",
		}
		if err := client.ProfileCreate("strict", config); err != nil {
			if strings.Contains(err.Error(), "exists") || client.ProfileExists("strict") {
				return nil
			}
			return fmt.Errorf("cannot create strict profile: %w", err)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
