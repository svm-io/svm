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

var ctxTimeout = 5 * time.Minute

var applySecureCmd = &cobra.Command{
	Use:   "apply-secure [vm-name...]",
	Short: "Apply security restrictions to existing VMs",
	Long: `Apply strict security configuration to one or all existing VMs.
If no VM name provided, applies to ALL existing VMs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
		defer cancel()

		client := lxd.New(ctx)

		if len(args) == 0 {
			return applyToAllVMs(ctx, client)
		}

		for _, vmName := range args {
			if err := applySecurityToVM(ctx, client, vmName); err != nil {
				return fmt.Errorf("failed to apply to %s: %w", vmName, err)
			}
		}
		return nil
	},
}

func applyToAllVMs(ctx context.Context, client *lxd.Client) error {
	vms, err := listAllVMs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}

	if len(vms) == 0 {
		fmt.Println("No VMs found")
		return nil
	}

	fmt.Printf("Found %d VMs, applying security restrictions...\n", len(vms))

	success := 0
	failed := 0
	for _, vm := range vms {
		fmt.Printf("  Processing %s... ", vm)
		if err := applySecurityToVM(ctx, client, vm); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			failed++
			continue
		}
		fmt.Printf("OK\n")
		success++
	}

	fmt.Printf("\nResults: %d applied, %d failed\n", success, failed)
	if failed > 0 {
		return fmt.Errorf("%d VMs failed", failed)
	}
	return nil
}

func applySecurityToVM(ctx context.Context, client *lxd.Client, name string) error {
	state, err := client.State(name)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	wasRunning := state == "Running"
	if wasRunning {
		if err := client.Stop(name); err != nil {
			return fmt.Errorf("failed to stop: %w", err)
		}
		fmt.Printf("(stopped) ")
	}

	if err := applyHostSecurityRestrictions(ctx, client, name); err != nil {
		return fmt.Errorf("host restrictions failed: %w", err)
	}
	fmt.Printf("(host secured) ")

	if err := applyGuestSecuritySetup(ctx, client, name); err != nil {
		fmt.Printf("(guest skip: %v) ", err)
	}

	if wasRunning {
		if err := client.Start(name); err != nil {
			return fmt.Errorf("failed to restart: %w", err)
		}
		fmt.Printf("(restarted)")
	}

	return nil
}

func applyHostSecurityRestrictions(ctx context.Context, client *lxd.Client, name string) error {
	restrictions := map[string]string{
		"security.nesting":       "false",
		"security.privileged":   "false",
		"security.protocols":    "clear",
		"linux.kernel_modules":    "",
		"security.devlxd":       "false",
		"security.idmap.base":  "0",
		"security.idmap.size":  "65536",
	}

	for key, value := range restrictions {
		if err := client.ConfigSet(name, key, value); err != nil {
			return err
		}
	}
	return nil
}

func applyGuestSecuritySetup(ctx context.Context, client *lxd.Client, name string) error {
	guestSetup := []string{
		"echo 'kernel.dmesg.restrict=1' > /etc/sysctl.d/99-security.conf",
		"echo 'kernel.kptr_restrict=2' >> /etc/sysctl.d/99-security.conf",
		"echo 'kernel.yama.ptrace_scope=1' >> /etc/sysctl.d/99-security.conf",
		"sysctl -p /etc/sysctl.d/99-security.conf",
	}

	for _, c := range guestSetup {
		if _, err := client.ExecToString(name, "sh", "-c", c); err != nil {
			return fmt.Errorf("guest cmd failed: %w", err)
		}
	}
	return nil
}

func listAllVMs(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "lxc", "list", "--format", "csv", "-n")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var vms []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) > 0 {
			vms = append(vms, strings.TrimSpace(parts[0]))
		}
	}
	return vms, nil
}

func init() {
	rootCmd.AddCommand(applySecureCmd)
}