package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"vms/pkg/display"
	"vms/pkg/lxd"
)

var (
	detach  bool
	x11Flag bool
)

var launchCmd = &cobra.Command{
	Use:   "launch <vm-name> <app> [args...]",
	Short: "Launch GUI in VM with Wayland/X11 to host",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		vmName := args[0]
		program := args[1]

		client := lxd.New(ctx)

		state, err := client.State(vmName)
		if err != nil {
			return fmt.Errorf("VM not found: %w", err)
		}
		if state != "Running" {
			if err := client.Start(vmName); err != nil {
				return fmt.Errorf("failed to start VM: %w", err)
			}
			if err := client.WaitForRunning(vmName, 2*time.Minute); err != nil {
				return fmt.Errorf("VM did not start: %w", err)
			}
		}

		disp, err := display.Detect()
		if err != nil {
			return fmt.Errorf("no display detected: %w", err)
		}

		if disp.Type == "wayland" {
			if err := forwardWaylandSocket(ctx, vmName, disp.Socket); err != nil {
				return fmt.Errorf("socket forward failed: %w", err)
			}
		} else {
			if err := forwardX11Socket(ctx, vmName); err != nil {
				return fmt.Errorf("x11 forward failed: %w", err)
			}
		}

		env := disp.Env()

		if detach {
			go func() {
				client.Exec(vmName, env, "nohup", program)
			}()
			fmt.Printf("App %q launched in VM %q\n", program, vmName)
			return nil
		}

		return client.Exec(vmName, env, program)
	},
}

func forwardWaylandSocket(ctx context.Context, vmName, socketName string) error {
	client := lxd.New(ctx)
	socketPath, err := display.GetSocketPath(socketName)
	if err != nil {
		return err
	}
	remotePath := fmt.Sprintf("%s/run/user/1000/%s", vmName, socketName)
	return client.FilePush(socketPath, remotePath)
}

func forwardX11Socket(ctx context.Context, vmName string) error {
	client := lxd.New(ctx)
	authPath, err := display.GetAuthPath()
	if err != nil {
		return err
	}
	remotePath := fmt.Sprintf("%s/root/.Xauthority", vmName)
	return client.FilePush(authPath, remotePath)
}

func startVM(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "start", name)
	return cmd.Run()
}

func init() {
	launchCmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run detached")
	rootCmd.AddCommand(launchCmd)
}