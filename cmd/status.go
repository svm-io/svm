package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"vms/pkg/lxd"
)

var statusCmd = &cobra.Command{
	Use:   "status [vm-name]",
	Short: "Show VM security status",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		if len(args) == 0 {
			return listAllVMStatus(ctx)
		}
		return showVMStatus(ctx, args[0])
	},
}

func listAllVMStatus(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "list", "--format", "csv", "-n")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	fmt.Println("VM Security Status")
	fmt.Println("================")

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, ",")
		name := strings.TrimSpace(parts[0])
		fmt.Printf("\n%s:\n", name)
		showVMStatus(ctx, name)
	}
	return nil
}

func showVMStatus(ctx context.Context, name string) error {
	client := lxd.New(ctx)
	info, err := client.Info(name)
	if err != nil {
		return err
	}

	fmt.Printf("  Status: %s\n", info["Status"])

	securityKeys := []string{
		"security.nesting",
		"security.privileged",
		"security.protocols",
	}

	fmt.Println("  Security:")
	for _, key := range securityKeys {
		value, _ := client.ConfigGet(name, key)
		fmt.Printf("    %s: %s\n", key, value)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}