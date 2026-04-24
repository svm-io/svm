# VMS: A Secure LXD VM Manager

## What This Project Is

VMS is a command-line tool designed to manage LXD virtual machines with strong security isolation while maintaining the ability to run graphical applications from the VM and display them on the host desktop. The project was born from a simple need: I wanted to run potentially untrusted applications in a completely isolated environment, but I still needed those applications to show their GUI windows on my host machine, just as if they were running locally.

The core philosophy here is straightforward. The VM should be as locked down as possible, with no ability to escape or access host resources directly, but the display pipeline should remain open so that when you launch a GUI application inside the VM, that window appears on your host's desktop. This is the "display forwarding" concept, borrowed from how X11 forwarding has worked for decades, but updated for modern Wayland sessions.

## How It Works

The architecture separates concerns into two distinct sides: the host side and the guest side.

On the host side, the tool detects whether you're running a Wayland session or an X11 session by examining the environment variables and socket files in your runtime directory. For Wayland, it finds the socket named wayland-* in XDG_RUNTIME_DIR. For X11, it looks for the DISPLAY variable and the corresponding Xauthority file. This detection is automatic and transparent, you don't need to tell the tool which display server you're using.

On the guest side, when a VM is created or updated, the tool applies a comprehensive set of security restrictions. These include disabling nested virtualization, preventing privileged mode operations, clearing the allowed kernel modules, and setting up user ID mapping so that root inside the container doesn't map to root on the host. Additionally, the tool can configure kernel parameters inside the guest to restrict kernel pointers in dmesg output, limit ptrace capabilities, and protect against symlink attacks.

The display connection works by copying the socket or authority file from host to VM, then setting the appropriate environment variables inside the VM before launching the application. When gedit runs inside the VM with WAYLAND_DISPLAY set, it connects back to your host's Wayland compositor and renders its window there, despite running inside an isolated VM.

## Commands Available

The tool provides several commands for different tasks.

The create command launches a new VM with Ubuntu and immediately applies all security restrictions. You can specify a different image if needed, but ubuntu:noble is the default.

The apply-secure command is particularly useful for existing VMs. Run it with no arguments and it will find all your VMs and apply security hardening to each one. Or specify VM names as arguments to target specific machines.

The launch command starts a GUI application inside a named VM with display forwarding to your host. This is how you actually use the isolated environment to run applications.

The test command runs an integration test suite that verifies display detection, LXD connectivity, image availability, VM creation with security settings, application installation and launch, and graphical output detection.

The status command shows the current security configuration of your VMs, displaying whether each security setting is properly applied.

The host-setup command prepares your host system for display forwarding by setting appropriate permissions on the Wayland and X11 sockets.

The guest-setup command configures security hardening inside a running VM, applying kernel parameters and system restrictions.

## Security Model

The security model here relies on several layers of defense.

At the hypervisor level, LXD provides isolation between VMs. Your VM runs its own kernel, completely separate from the host kernel, which means even if an attacker exploits a kernel vulnerability inside the VM, they still need to escape the hypervisor to reach the host.

At the LXD configuration level, we disable nesting, which prevents the VM from creating its own containers. We disable privileged mode so the VM cannot perform operations that would give it root access on the host. We clear kernel modules to prevent loading potentially dangerous code. And we map user IDs to a range that doesn't overlap with host users.

At the guest OS level inside the VM, we apply kernel parameters that restrict information disclosure through dmesg and hide kernel pointers that could aid in exploitation. We restrict ptrace to prevent one process from inspecting another without consent.

The display pipeline is intentionally left open because that's the entire point of the project. However, Wayland provides some inherent protection because applications can only render to windows they create, they cannot intercept or inject events into other applications, and the compositor mediates all display operations. X11 is less secure by design, which is why Wayland is preferred.

## Building and Running

To build the project, ensure you have Go 1.21 or later installed, then run go build from the project directory. This produces the vms binary.

Before using the tool, make sure LXD is installed and initialized on your system with a storage pool configured. You'll need either a Wayland session or an X11 session running on the host for display forwarding to work.

For a quick test of your setup, run ./vms test which runs through the integration test suite and reports what works and what doesn't on your system.

## Architecture Notes

The codebase is organized into clear layers. The cmd package contains CLI commands built with Cobra, each command is its own file. The pkg/lxd package wraps the lxc command-line tool with a Go API. The pkg/display package handles display detection and configuration. The config directory contains configuration templates.

This organization keeps concerns separated. Commands don't directly call lxc, they use the client. Display detection is isolated in its own package and can be tested independently.

## The Road Ahead

The project has matured to a functional state, but there's always more to do. A profile system for different security levels would allow quick switching between strict, standard, and development modes. Container support alongside VM support would provide lighter-weight isolation when full virtualization isn't needed. Snapshot and rollback functionality would enable clean recovery after testing potentially destructive applications. A daemon mode would allow the tool to run in the background and manage applications automatically. And integration with systems like snapd or flatpak would provide application sandboxing at higher levels.

The core idea is solid though. Run untrusted code in a VM that can't escape, but still see what it's doing on your desktop. That's a useful pattern, and this tool makes it easy.