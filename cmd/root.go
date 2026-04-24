package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vms",
	Short: "Secure LXD VM launcher with display forwarding",
	Long:  `Launch GUI apps in isolated LXD VMs with Wayland/X11 to host.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}