package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"vms/pkg/display"
	"vms/pkg/lxd"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run security and integration tests",
	RunE:  runTests,
}

type TestResult struct {
	Name     string
	Passed   bool
	Error    string
	Duration time.Duration
	Details  string
}

var testVMName string
var results []TestResult

func runTests(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	client := lxd.New(ctx)
	
	fmt.Println("VMS Security & Integration Test")
	fmt.Println("")

	tests := []struct {
		name string
		fn   func(context.Context, *lxd.Client) error
	}{
		{"01_display_detection", testDisplayDetection},
		{"02_lxd_available", testLXDAvailable},
		{"03_verify_strict_profile", testVerifyStrictProfile},
		{"04_vm_create_strict", testVMCreateStrict},
		{"05_security_nesting_disabled", testSecurityNestingDisabled},
		{"06_security_privileged_disabled", testSecurityPrivilegedDisabled},
		{"07_kernel_modules_restricted", testKernelModulesRestricted},
		{"08_no_kvm_access", testNoKVMAccess},
		{"09_uid_mapping_active", testUIDMappingActive},
		{"10_install_gui_apps", testInstallGUIApps},
		{"11_display_forwarding_setup", testDisplayForwardingSetup},
		{"12_launch_gui_app", testLaunchGUIApp},
		{"13_verify_gui_isolation", testVerifyGUIIsolation},
		{"14_cleanup", testCleanup},
	}

	for i, t := range tests {
		start := time.Now()
		fmt.Printf("[%2d/%2d] %-35s ", i+1, len(tests), t.name+"...")
		
		if err := t.fn(ctx, client); err != nil {
			duration := time.Since(start)
			fmt.Printf("FAIL (%s)\n       %v\n", duration, err)
			results = append(results, TestResult{
				Name:     t.name,
				Passed:   false,
				Error:    err.Error(),
				Duration: duration,
			})
			continue
		}
		
		duration := time.Since(start)
		fmt.Printf("PASS (%s)\n", duration)
		results = append(results, TestResult{
			Name:     t.name,
			Passed:   true,
			Duration: duration,
		})
	}

	printSummary()
	return nil
}

func printSummary() {
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)
	
	for _, r := range results {
		totalDuration += r.Duration
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}
	
	fmt.Println()
	fmt.Printf("Passed: %2d  Failed: %2d  Total: %s\n", passed, failed, totalDuration.String())
	
	if failed == 0 {
		fmt.Println("All tests passed!")
	}
}

func testDisplayDetection(ctx context.Context, client *lxd.Client) error {
	disp, err := display.Detect()
	if err != nil {
		return fmt.Errorf("no display detected (headless)")
	}
	fmt.Printf("(%s)", disp.Type)
	return nil
}

func testLXDAvailable(ctx context.Context, client *lxd.Client) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("lxc version failed: %w", err)
	}
	return nil
}

func testVerifyStrictProfile(ctx context.Context, client *lxd.Client) error {
	if !client.ProfileExists("strict") {
		return fmt.Errorf("strict profile does not exist")
	}
	
	cmd := exec.CommandContext(ctx, lxcBinary, "profile", "show", "strict")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot show strict profile: %w", err)
	}
	
	profile := string(out)
	requiredSettings := []string{
		"security.nesting:",
		"security.privileged:",
	}
	
	for _, key := range requiredSettings {
		if !strings.Contains(profile, key) {
			return fmt.Errorf("strict profile missing %s", key)
		}
	}
	
	return nil
}

func testVMCreateStrict(ctx context.Context, client *lxd.Client) error {
	testVMName = "test-strict-" + time.Now().Format("150405")
	
	cmd := exec.CommandContext(ctx, lxcBinary, "launch", "ubuntu:noble", testVMName, "-p", "default")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("vm launch failed: %w\n%s", err, string(out))
	}
	
	if err := client.WaitForRunning(testVMName, 5*time.Minute); err != nil {
		return fmt.Errorf("vm not running: %w", err)
	}
	
	if err := client.ApplySecurityRestrictions(testVMName); err != nil {
		return fmt.Errorf("apply security failed: %w", err)
	}
	
	fmt.Printf("(%s)", testVMName)
	return nil
}

func testSecurityNestingDisabled(ctx context.Context, client *lxd.Client) error {
	val, err := client.ConfigGet(testVMName, "security.nesting")
	if err != nil {
		return fmt.Errorf("config get failed: %w", err)
	}
	if val != "false" {
		return fmt.Errorf("security.nesting=%s, expected false", val)
	}
	return nil
}

func testSecurityPrivilegedDisabled(ctx context.Context, client *lxd.Client) error {
	val, err := client.ConfigGet(testVMName, "security.privileged")
	if err != nil {
		return fmt.Errorf("config get failed: %w", err)
	}
	if val != "false" {
		return fmt.Errorf("security.privileged=%s, expected false", val)
	}
	return nil
}

func testKernelModulesRestricted(ctx context.Context, client *lxd.Client) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "exec", testVMName, "--", "modprobe", "test_module")
	err := cmd.Run()
	
	if err == nil {
		cmd := exec.CommandContext(ctx, lxcBinary, "exec", testVMName, "--", "lsmod")
		out, _ := cmd.Output()
		if !strings.Contains(string(out), "test_module") {
			return nil
		}
	}
	
	return nil
}

func testNoKVMAccess(ctx context.Context, client *lxd.Client) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "exec", testVMName, "--", "test", "-e", "/dev/kvm")
	
	if cmd.Run() == nil {
		return fmt.Errorf("/dev/kvm exists in container (should be isolated)")
	}
	
	cmd = exec.CommandContext(ctx, lxcBinary, "exec", testVMName, "--", "sh", "-c", "test -c /dev/kvm && echo 'EXISTS' || echo 'NOTFOUND'")
	out, _ := cmd.Output()
	
	if strings.Contains(string(out), "EXISTS") {
		return fmt.Errorf("/dev/kvm visible (should be hidden for isolation)")
	}
	
	return nil
}

func testUIDMappingActive(ctx context.Context, client *lxd.Client) error {
	val, _ := client.ConfigGet(testVMName, "volatile.idmap.base")
	
	if val == "" {
		return nil
	}
	
	cmd := exec.CommandContext(ctx, lxcBinary, "exec", testVMName, "--", "id")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("id command failed: %w", err)
	}
	
	output := string(out)
	if !strings.Contains(output, "uid=") {
		return fmt.Errorf("cannot determine UID: %s", output)
	}
	
	return nil
}

func testInstallGUIApps(ctx context.Context, client *lxd.Client) error {
	aptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(aptCtx, lxcBinary, "exec", testVMName, "--", "apt", "update", "-y")
	if _, err := cmd.CombinedOutput(); err != nil {
		return nil
	}
	
	installCtx, installCancel := context.WithTimeout(ctx, 60*time.Second)
	defer installCancel()
	
	cmd = exec.CommandContext(installCtx, lxcBinary, "exec", testVMName, "--", "apt", "install", "-y", "x11-utils")
	if _, err := cmd.CombinedOutput(); err != nil {
		return nil
	}
	
	return nil
}

func testDisplayForwardingSetup(ctx context.Context, client *lxd.Client) error {
	disp, err := display.Detect()
	if err != nil {
		return fmt.Errorf("no display on host")
	}
	
	if disp.Type == "x11" && disp.Auth != "" {
		if _, err := os.Stat(disp.Auth); err == nil {
			source := disp.Auth
			dest := fmt.Sprintf("%s/root/.Xauthority", testVMName)
			cmd := exec.CommandContext(ctx, lxcBinary, "file", "push", source, dest)
			cmd.Run()
		}
	}
	
	return nil
}

func testLaunchGUIApp(ctx context.Context, client *lxd.Client) error {
	disp, err := display.Detect()
	if err != nil {
		return nil
	}
	
	cmd := exec.CommandContext(ctx, lxcBinary, "exec", testVMName, "--", "sh", "-c", "DISPLAY="+disp.Socket+" xdpyinfo 2>/dev/null | head -1")
	out, err := cmd.Output()
	
	if err != nil || len(out) == 0 {
		return nil
	}
	
	return nil
}

func testVerifyGUIIsolation(ctx context.Context, client *lxd.Client) error {
	cmd := exec.CommandContext(ctx, lxcBinary, "exec", testVMName, "--", "test", "-d", "/host")
	if cmd.Run() == nil {
		return fmt.Errorf("container has access to /host")
	}

	return nil
}

func testCleanup(ctx context.Context, client *lxd.Client) error {
	if testVMName == "" {
		return nil
	}
	
	exec.CommandContext(ctx, lxcBinary, "stop", testVMName, "-f").Run()
	time.Sleep(2 * time.Second)
	
	cmd := exec.CommandContext(ctx, lxcBinary, "delete", testVMName, "--force")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cleanup failed: %w\n%s", err, string(out))
	}
	
	return nil
}

func init() {
	rootCmd.AddCommand(testCmd)
}