# Setup Scripts

This directory contains convenient setup scripts for configuring vms environments.

## setup-db-vm.sh

Creates a production-ready Fedora VM with complete security hardening and display forwarding capabilities.

### Usage

```bash
./scripts/setup-db-vm.sh [vm-name] [fedora-version]
```

### Parameters

- `vm-name` (optional, default: "db"): Name of the VM to create
- `fedora-version` (optional, default: "40"): Fedora version to use

### Examples

```bash
# Create default 'db' VM with Fedora 40
./scripts/setup-db-vm.sh

# Create custom VM with specific Fedora version
./scripts/setup-db-vm.sh myapp 39

# Create multiple isolated VMs
./scripts/setup-db-vm.sh db-prod 40
./scripts/setup-db-vm.sh db-test 40
./scripts/setup-db-vm.sh db-dev 40
```

### What It Does

1. **Creates VM** with specified Fedora image from LXD image server
2. **Applies Security** with full isolation: nesting disabled, privileged mode off, kernel module restrictions
3. **Hardens Guest** with kernel security parameters (dmesg, ptrace, kptr restrictions)
4. **Verifies Status** to ensure proper configuration

### Next Steps

After setup, launch GUI applications in your VM:

```bash
./vms launch db gedit
./vms launch db firefox
./vms launch db xterm
```

Applications appear on your host display with Wayland or X11, while the VM remains completely isolated from the rest of your system.
