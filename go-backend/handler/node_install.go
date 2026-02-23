package handler

import (
	"flux-panel/go-backend/config"
	"flux-panel/go-backend/model"
	"flux-panel/go-backend/service"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func NodeInstallScript(c *gin.Context) {
	script := `#!/bin/bash
set -e

NODE_ID=$1
NODE_SECRET=$2
PANEL_ADDR=$3
USE_IPV6=$4

if [ -z "$NODE_ID" ] || [ -z "$NODE_SECRET" ] || [ -z "$PANEL_ADDR" ]; then
    echo "Usage: $0 <node_id> <node_secret> <panel_addr> [6]"
    exit 1
fi

CURL_FLAGS="-fsSL"
if [ "$USE_IPV6" = "6" ]; then
    CURL_FLAGS="-6fsSL"
    echo "IPv6 mode enabled"
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Stop existing service if running (avoid "text file busy" on overwrite)
if systemctl is-active --quiet gost-node 2>/dev/null; then
    echo "Stopping existing gost-node service..."
    systemctl stop gost-node
fi
rm -f /usr/local/bin/gost-node

# Download binary
echo "Downloading gost-node for $ARCH..."
curl $CURL_FLAGS "$PANEL_ADDR/node-install/binary/$ARCH" -o /usr/local/bin/gost-node
chmod +x /usr/local/bin/gost-node

# Create data directory
mkdir -p /etc/gost

# Install Xray from panel
if [ -x /usr/local/bin/xray ]; then
    echo "Xray already installed, skipping..."
else
    echo "Installing Xray for $ARCH..."
    curl $CURL_FLAGS "$PANEL_ADDR/node-install/xray/$ARCH" -o /usr/local/bin/xray || { echo "Warning: Xray download failed, skipping"; }
    if [ -f /usr/local/bin/xray ]; then
        chmod +x /usr/local/bin/xray
        cp /usr/local/bin/xray /etc/gost/xray
        echo "Xray installed: $(/usr/local/bin/xray version 2>/dev/null | head -1)"
    fi
fi

# Detect TLS from panel address
USE_TLS=false
ADDR_VALUE="$PANEL_ADDR"
case "$ADDR_VALUE" in
    https://*) USE_TLS=true ;;
esac
ADDR_VALUE="${ADDR_VALUE#http://}"
ADDR_VALUE="${ADDR_VALUE#https://}"
ADDR_VALUE="${ADDR_VALUE%/}"

# Generate config.json
cat > /etc/gost/config.json << EOF
{
  "addr": "$ADDR_VALUE",
  "secret": "$NODE_SECRET",
  "use_tls": $USE_TLS
}
EOF

# Ensure runtime config file exists
if [ ! -f /etc/gost/gost.json ]; then
    echo "{}" > /etc/gost/gost.json
fi

# Create systemd service
cat > /etc/systemd/system/gost-node.service << EOF
[Unit]
Description=GOST Node
After=network.target

[Service]
Type=simple
WorkingDirectory=/etc/gost
ExecStart=/usr/local/bin/gost-node
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable gost-node
systemctl restart gost-node

echo "GOST Node installed and started successfully!"
echo "Node ID: $NODE_ID"
echo "Config: /etc/gost/config.json"
echo "Logs: journalctl -u gost-node -f"
`
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, script)
}

var allowedArchs = map[string]bool{
	"amd64": true,
	"arm64": true,
	"arm":   true,
}

func NodeInstallBinary(c *gin.Context) {
	arch := c.Param("arch")
	if !allowedArchs[arch] {
		c.String(http.StatusBadRequest, "invalid architecture")
		return
	}

	binaryPath := filepath.Join(config.Cfg.NodeBinaryDir, fmt.Sprintf("node-%s", arch))

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		c.String(http.StatusNotFound, "Binary not found for architecture: "+arch)
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=node-%s", arch))
	c.File(binaryPath)
}

func NodeUninstallScript(c *gin.Context) {
	script := `#!/bin/bash
set -e

echo "Stopping gost-node service..."
systemctl stop gost-node 2>/dev/null || true
systemctl disable gost-node 2>/dev/null || true
rm -f /etc/systemd/system/gost-node.service
systemctl daemon-reload

echo "Removing binaries..."
rm -f /usr/local/bin/gost-node
rm -f /usr/local/bin/xray

echo "Removing config directory..."
rm -rf /etc/gost

echo "GOST Node uninstalled successfully!"
`
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, script)
}

func NodeInstallXray(c *gin.Context) {
	arch := c.Param("arch")
	if !allowedArchs[arch] {
		c.String(http.StatusBadRequest, "invalid architecture")
		return
	}

	binaryPath := filepath.Join(config.Cfg.NodeBinaryDir, fmt.Sprintf("xray-%s", arch))

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		c.String(http.StatusNotFound, "Xray binary not found for architecture: "+arch)
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=svc-%s", arch))
	c.File(binaryPath)
}

// ─── Camouflaged install handlers ───

// findNodeBySecret looks up a node by its secret.
func findNodeBySecret(secret string) *model.Node {
	var node model.Node
	if err := service.DB.Where("secret = ?", secret).First(&node).Error; err != nil {
		return nil
	}
	return &node
}

// CamoInstallScript generates a fully pre-configured install script with disguised paths.
// The secret in the URL path serves as authentication.
func CamoInstallScript(c *gin.Context) {
	secret := c.Param("secret")
	node := findNodeBySecret(secret)
	if node == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	// Ensure disguise names exist (backfill for legacy nodes)
	disguise := node.DisguiseName
	xrayDisguise := node.XrayDisguiseName
	if disguise == "" {
		disguise = "gost-node"
	}
	if xrayDisguise == "" {
		xrayDisguise = "xray"
	}

	panelAddr := service.GetPanelAddress(c.GetHeader("Origin"))

	// Detect TLS from panel address
	useTLS := strings.HasPrefix(panelAddr, "https://")

	// Strip scheme for config addr
	addrValue := panelAddr
	addrValue = strings.TrimPrefix(addrValue, "http://")
	addrValue = strings.TrimPrefix(addrValue, "https://")
	addrValue = strings.TrimSuffix(addrValue, "/")

	script := fmt.Sprintf(`#!/bin/bash
set -e

# ─── Camouflaged Node Install Script ───
DISGUISE="%s"
XRAY_DISGUISE="%s"
NODE_SECRET="%s"
PANEL_ADDR="%s"
USE_TLS=%t

CURL_FLAGS="-fsSL"
if [ "${1}" = "6" ]; then
    CURL_FLAGS="-6fsSL"
    echo "IPv6 mode enabled"
fi

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Stop existing service if running
if systemctl is-active --quiet "$DISGUISE" 2>/dev/null; then
    echo "Stopping existing $DISGUISE service..."
    systemctl stop "$DISGUISE"
fi
# Also stop legacy gost-node service if present (migration)
if systemctl is-active --quiet gost-node 2>/dev/null; then
    echo "Stopping legacy service..."
    systemctl stop gost-node
    systemctl disable gost-node 2>/dev/null || true
    rm -f /etc/systemd/system/gost-node.service
fi
rm -f "/usr/local/bin/$DISGUISE"

# Download binary
echo "Downloading service binary for $ARCH..."
curl $CURL_FLAGS "$PANEL_ADDR/s/$NODE_SECRET/b/$ARCH" -o "/usr/local/bin/$DISGUISE"
chmod +x "/usr/local/bin/$DISGUISE"

# Create config directory
mkdir -p "/etc/$DISGUISE"

# Install secondary binary
if [ -x "/usr/local/bin/$XRAY_DISGUISE" ]; then
    echo "Secondary binary already installed, skipping..."
else
    echo "Installing secondary binary for $ARCH..."
    curl $CURL_FLAGS "$PANEL_ADDR/s/$NODE_SECRET/x/$ARCH" -o "/usr/local/bin/$XRAY_DISGUISE" || { echo "Warning: Secondary binary download failed, skipping"; }
    if [ -f "/usr/local/bin/$XRAY_DISGUISE" ]; then
        chmod +x "/usr/local/bin/$XRAY_DISGUISE"
        cp "/usr/local/bin/$XRAY_DISGUISE" "/etc/$DISGUISE/$XRAY_DISGUISE"
        echo "Secondary binary installed"
    fi
fi

# Strip scheme for config addr
ADDR_VALUE="%s"

# Generate config.json
cat > "/etc/$DISGUISE/config.json" << EOF
{
  "addr": "$ADDR_VALUE",
  "secret": "$NODE_SECRET",
  "use_tls": $USE_TLS,
  "v_bin": "$XRAY_DISGUISE",
  "v_cfg": "service.json"
}
EOF

# Ensure runtime config file exists
if [ ! -f "/etc/$DISGUISE/runtime.json" ]; then
    echo "{}" > "/etc/$DISGUISE/runtime.json"
fi

# Create systemd service
cat > "/etc/systemd/system/$DISGUISE.service" << EOF
[Unit]
Description=$DISGUISE daemon
After=network.target

[Service]
Type=simple
WorkingDirectory=/etc/$DISGUISE
ExecStart=/usr/local/bin/$DISGUISE
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Create uninstall script
cat > "/etc/$DISGUISE/uninstall.sh" << 'UNINSTALL'
#!/bin/bash
set -e
DISGUISE="%s"
XRAY_DISGUISE="%s"
echo "Stopping service..."
systemctl stop "$DISGUISE" 2>/dev/null || true
systemctl disable "$DISGUISE" 2>/dev/null || true
rm -f "/etc/systemd/system/$DISGUISE.service"
systemctl daemon-reload
echo "Removing binaries..."
rm -f "/usr/local/bin/$DISGUISE"
rm -f "/usr/local/bin/$XRAY_DISGUISE"
echo "Removing config directory..."
rm -rf "/etc/$DISGUISE"
echo "Uninstalled successfully!"
UNINSTALL
chmod +x "/etc/$DISGUISE/uninstall.sh"

systemctl daemon-reload
systemctl enable "$DISGUISE"
systemctl restart "$DISGUISE"

echo "Service installed and started successfully!"
echo "Config: /etc/$DISGUISE/config.json"
echo "Logs: journalctl -u $DISGUISE -f"
echo "Uninstall: bash /etc/$DISGUISE/uninstall.sh"
`, disguise, xrayDisguise, node.Secret, panelAddr, useTLS, addrValue, disguise, xrayDisguise)

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, script)
}

// CamoInstallBinary serves the gost-node binary via camouflaged URL.
func CamoInstallBinary(c *gin.Context) {
	secret := c.Param("secret")
	node := findNodeBySecret(secret)
	if node == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	arch := c.Param("arch")
	if !allowedArchs[arch] {
		c.String(http.StatusBadRequest, "invalid architecture")
		return
	}

	binaryPath := filepath.Join(config.Cfg.NodeBinaryDir, fmt.Sprintf("node-%s", arch))
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		c.String(http.StatusNotFound, "binary not found")
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=bin-%s", arch))
	c.File(binaryPath)
}

// CamoInstallXray serves the xray binary via camouflaged URL.
func CamoInstallXray(c *gin.Context) {
	secret := c.Param("secret")
	node := findNodeBySecret(secret)
	if node == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	arch := c.Param("arch")
	if !allowedArchs[arch] {
		c.String(http.StatusBadRequest, "invalid architecture")
		return
	}

	binaryPath := filepath.Join(config.Cfg.NodeBinaryDir, fmt.Sprintf("xray-%s", arch))
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		c.String(http.StatusNotFound, "binary not found")
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=svc-%s", arch))
	c.File(binaryPath)
}
