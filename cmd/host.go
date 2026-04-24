package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"vms/pkg/lxd"
)

var hostSetupCmd = &cobra.Command{
	Use:   "host-setup",
	Short: "Setup host security for LXD display forwarding",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return runHostSetup(ctx)
	},
}

func runHostSetup(ctx context.Context) error {
	fmt.Println("Setting up host for display forwarding...")

	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime == "" {
		uid := os.Getuid()
		xdgRuntime = fmt.Sprintf("/run/user/%d", uid)
	}

	setupCmds := []struct {
		name string
		argv []string
	}{
		{"Wayland socket permissions", []string{"chmod", "755", xdgRuntime}},
		{"X11 socket permissions", []string{"chmod", "1777", "/tmp/.X11-unix"}},
		{"DRI device access", []string{"sh", "-c", "chmod 700 /dev/dri/* 2>/dev/null || true"}},
	}

	for _, s := range setupCmds {
		fmt.Printf("  %s... ", s.name)
		cmd := exec.CommandContext(ctx, s.argv[0], s.argv[1:]...)
		if err := cmd.Run(); err != nil {
			fmt.Printf("skip (may need sudo)\n")
			continue
		}
		fmt.Println("OK")
	}

	return nil
}

var guestSetupCmd = &cobra.Command{
	Use:   "guest-setup <vm-name>",
	Short: "Setup guest security inside VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return runGuestSetup(ctx, args[0])
	},
}

func runGuestSetup(ctx context.Context, name string) error {
	fmt.Printf("Setting up guest security for %s...\n", name)

	client := lxd.New(ctx)

	guestCmds := []struct {
		name string
		cmd  string
	}{
		{"Restrict dmesg", "sysctl -w kernel.dmesg.restrict=1"},
		{"Restrict kernel pointers", "sysctl -w kernel.kptr_restrict=2"},
		{"Ptrace restrictions", "sysctl -w kernel.yama.ptrace_scope=1"},
		{"Disable core dumps", "ulimit -c 0"},
		{"Secure symlinks", "sysctl -w fs.protected_symlinks=1"},
		{"Secure hardlinks", "sysctl -w fs.protected_hardlinks=1"},
	}

	for _, s := range guestCmds {
		fmt.Printf("  %s... ", s.name)
		_, err := client.ExecToString(name, "sh", "-c", s.cmd)
		if err != nil {
			fmt.Printf("skip\n")
			continue
		}
		fmt.Println("OK")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(hostSetupCmd)
	rootCmd.AddCommand(guestSetupCmd)
}