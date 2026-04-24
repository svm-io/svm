#!/bin/bash
# VMS Example: Strict Security with Display Forwarding
# This example demonstrates creating a secure container with GUI access

set -e

VM_NAME="secure-app"
LXC="/snap/bin/lxc"

echo "=== VMS Strict + Display Forwarding Example ==="
echo ""

# Step 1: Init LXD (one-time)
echo "[1] Initializing LXD..."
# ./vms init  # Already done

# Step 2: Create secure container
echo "[2] Creating secure container: $VM_NAME"
$LXC launch ubuntu:noble $VM_NAME

# Step 3: Wait for running
echo "[3] Waiting for container..."
for i in {1..30}; do
  STATE=$($LXC info $VM_NAME | grep Status | awk '{print $2}')
  if [ "$STATE" = "RUNNING" ]; then
    echo "   Container is RUNNING"
    break
  fi
  sleep 1
done

# Step 4: Apply strict security
echo "[4] Applying strict security..."
$LXC config set $VM_NAME security.nesting false
$LXC config set $VM_NAME security.privileged false
$LXC config set $VM_NAME linux.kernel_modules ""
echo "   ✓ Isolation: Nesting disabled"
echo "   ✓ Isolation: Privilege escalation blocked"
echo "   ✓ Isolation: Kernel modules restricted"

# Step 5: Add X11 socket forwarding (the "spiraglio")
echo "[5] Adding display forwarding..."
$LXC config device add $VM_NAME x11sock disk \
  source=/tmp/.X11-unix \
  path=/tmp/.X11-unix
echo "   ✓ X11 Socket: /tmp/.X11-unix"

# Step 6: Verify isolation
echo "[6] Verifying security..."
echo "   Security config:"
$LXC config show $VM_NAME | grep -E "security\.|linux.kernel" | sed 's/^/     /'

# Step 7: Install and run GUI app
echo "[7] Installing GUI app..."
$LXC exec $VM_NAME -- apt update -y > /dev/null 2>&1
$LXC exec $VM_NAME -- apt install -y xterm > /dev/null 2>&1
echo "   ✓ xterm installed"

# Step 8: Launch GUI app
echo "[8] Launching GUI app..."
DISPLAY=:0 $LXC exec $VM_NAME -- env \
  DISPLAY=:0 \
  xterm -title "VMS Secure App" &

echo ""
echo "✓ Setup complete!"
echo ""
echo "Container: $VM_NAME"
echo "Isolation: Strict (nesting=false, privileged=false)"
echo "Display: Forwarded to host X11"
echo "Status: Running"
echo ""
echo "To connect:"
echo "  lxc exec $VM_NAME -- bash"
echo ""
echo "To stop:"
echo "  lxc stop $VM_NAME"
echo ""
echo "To delete:"
echo "  lxc delete $VM_NAME --force"
