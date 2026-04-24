package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
		programArgs := args[2:]

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

		if x11Flag {
			disp.Type = "x11"
		}

		var env []string
		if disp.Type == "wayland" {
			env, err = setupWaylandForward(ctx, client, vmName, disp.Socket)
			if err != nil {
				return fmt.Errorf("wayland setup failed: %w", err)
			}
		} else {
			env, err = setupX11Forward(ctx, client, vmName)
			if err != nil {
				return fmt.Errorf("x11 setup failed: %w", err)
			}
		}

		env = append(env, disp.Env()...)

		fullCmd := append([]string{program}, programArgs...)
		if detach {
			go func() {
				client.Exec(vmName, env, append([]string{"nohup"}, fullCmd...)...)
			}()
			fmt.Printf("App %q launched in VM %q (detached)\n", program, vmName)
			return nil
		}

		return client.Exec(vmName, env, fullCmd...)
	},
}

func setupWaylandForward(ctx context.Context, client *lxd.Client, vmName, socketName string) ([]string, error) {
	socketPath, err := display.GetSocketPath(socketName)
	if err != nil {
		return nil, err
	}

	remoteDir := fmt.Sprintf("%s/run/user/1000", vmName)
	if err := client.ExecToString(vmName, "mkdir", "-p", remoteDir); err != nil {
		return nil, fmt.Errorf("failed to create runtime dir: %w", err)
	}

	remotePath := filepath.Join(remoteDir, filepath.Base(socketPath))
	if err := client.FilePush(socketPath, remotePath); err != nil {
		return nil, fmt.Errorf("socket forward failed: %w", err)
	}

	if err := client.ExecToString(vmName, "chmod", "660", remotePath); err != nil {
		return nil, fmt.Errorf("chmod failed: %w", err)
	}

	uidOutput, _ := client.ExecToString(vmName, "id", "-u")
	gidOutput, _ := client.ExecToString(vmName, "id", "-g")
	uid := "1000"
	gid := "1000"
	if uidOutput != "" {
		uid = strings.TrimSpace(uidOutput)
	}
	if gidOutput != "" {
		gid = strings.TrimSpace(gidOutput)
	}

	if err := client.ExecToString(vmName, "chown", uid+":"+gid, remotePath); err != nil {
		return nil, fmt.Errorf("chown failed: %w", err)
	}

	return []string{
		fmt.Sprintf("XDG_RUNTIME_DIR=/run/user/1000"),
		fmt.Sprintf("WAYLAND_DISPLAY=%s", filepath.Base(socketPath)),
	}, nil
}

func setupX11Forward(ctx context.Context, client *lxd.Client, vmName string) ([]string, error) {
	authPath, err := display.GetAuthPath()
	if err != nil {
		return nil, err
	}

	remoteDir := fmt.Sprintf("%s/root", vmName)
	if err := client.ExecToString(vmName, "mkdir", "-p", remoteDir); err != nil {
		return nil, fmt.Errorf("failed to create home dir: %w", err)
	}

	remoteAuthPath := filepath.Join(remoteDir, ".Xauthority")
	if err := client.FilePush(authPath, remoteAuthPath); err != nil {
		return nil, fmt.Errorf("xauth forward failed: %w", err)
	}

	if err := client.ExecToString(vmName, "chmod", "600", remoteAuthPath); err != nil {
		return nil, fmt.Errorf("chmod failed: %w", err)
	}

	return []string{
		fmt.Sprintf("DISPLAY=%s", os.Getenv("DISPLAY")),
		fmt.Sprintf("XAUTHORITY=%s", remoteAuthPath),
	}, nil
}

func startVM(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "start", name)
	return cmd.Run()
}

func init() {
	launchCmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run detached")
	launchCmd.Flags().BoolVarP(&x11Flag, "x11", "x", false, "Force X11 mode")
	rootCmd.AddCommand(launchCmd)
}