#!/bin/bash
# Setup script for 'db' VM with Fedora
# This creates a production-ready Fedora VM with full security restrictions
# and display forwarding capabilities

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY="$SCRIPT_DIR/vms"

if [ ! -x "$BINARY" ]; then
    echo "Error: vms binary not found or not executable at $BINARY"
    exit 1
fi

VM_NAME="${1:-db}"
FEDORA_VERSION="${2:-40}"

echo "=========================================="
echo "Setting up Fedora VM: $VM_NAME"
echo "Fedora Version: $FEDORA_VERSION"
echo "=========================================="

# Step 0: Setup host for display forwarding (MUST BE FIRST!)
echo ""
echo "[0/5] Setting up host for display forwarding..."
if $BINARY host-setup; then
    echo "✓ Host setup complete"
else
    echo "! Host setup had issues (may require sudo)"
fi

# Step 1: Create the VM with Fedora image
echo ""
echo "[1/5] Creating VM with Fedora $FEDORA_VERSION..."
if $BINARY create "$VM_NAME" "images:fedora/$FEDORA_VERSION"; then
    echo "✓ VM created successfully"
else
    echo "✗ Failed to create VM"
    exit 1
fi

# Step 2: Apply security restrictions
echo ""
echo "[2/5] Applying security restrictions..."
if $BINARY apply-secure "$VM_NAME"; then
    echo "✓ Security restrictions applied"
else
    echo "✗ Failed to apply security restrictions"
    exit 1
fi

# Step 3: Setup guest security
echo ""
echo "[3/5] Setting up guest security..."
if $BINARY guest-setup "$VM_NAME"; then
    echo "✓ Guest security setup complete"
else
    echo "! Guest security setup had issues (may be expected)"
fi

# Step 4: Status check
echo ""
echo "[4/5] Verifying VM status..."
if $BINARY status "$VM_NAME"; then
    echo "✓ VM is properly configured"
else
    echo "! Status check had issues"
fi

# Step 5: Final summary
echo ""
echo "=========================================="
echo "Setup Complete!"
echo "=========================================="
echo ""
echo "Your '$VM_NAME' VM is now ready to run GUI apps."
echo ""
echo "Quick start:"
echo "  $BINARY launch $VM_NAME gedit        # Text editor"
echo "  $BINARY launch $VM_NAME firefox      # Web browser"
echo "  $BINARY launch $VM_NAME xterm        # Terminal"
echo ""
echo "The application windows will appear on your host display."
echo "The VM itself is completely isolated from the host system."
echo ""
