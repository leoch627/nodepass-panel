#!/bin/bash

# Detect system architecture
get_architecture() {
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            echo "amd64"
            ;;
    esac
}

INSTALL_DIR="/etc/flux-node"
SERVICE_NAME="flux-node"
BINARY_NAME="flux-node"

# Build download URL from panel address
build_download_url() {
    local ARCH=$(get_architecture)
    local BASE_URL="$SERVER_ADDR"
    if [[ ! "$BASE_URL" =~ ^https?:// ]]; then
        BASE_URL="http://${BASE_URL}"
    fi
    BASE_URL="${BASE_URL%/}"
    echo "${BASE_URL}/node-install/binary/${ARCH}"
}

# Load panel address from existing config (for update/uninstall)
load_existing_config() {
    if [[ -f "$INSTALL_DIR/config.json" && -z "$SERVER_ADDR" ]]; then
        SERVER_ADDR=$(grep -o '"addr"[[:space:]]*:[[:space:]]*"[^"]*"' "$INSTALL_DIR/config.json" | sed 's/"addr"[[:space:]]*:[[:space:]]*"//;s/"$//')
    fi
}

show_menu() {
  echo "==============================================="
  echo "            Node Management Script"
  echo "==============================================="
  echo "Select an action:"
  echo "1. Install"
  echo "2. Update"
  echo "3. Uninstall"
  echo "4. Exit"
  echo "==============================================="
}

delete_self() {
  echo ""
  echo "Cleaning up script file..."
  SCRIPT_PATH="$(readlink -f "$0" 2>/dev/null || realpath "$0" 2>/dev/null || echo "$0")"
  sleep 1
  rm -f "$SCRIPT_PATH" && echo "Script file deleted" || echo "Failed to delete script file"
}

check_and_install_tcpkill() {
  if command -v tcpkill &> /dev/null; then
    return 0
  fi

  OS_TYPE=$(uname -s)

  if [[ $EUID -ne 0 ]]; then
    SUDO_CMD="sudo"
  else
    SUDO_CMD=""
  fi

  if [[ "$OS_TYPE" == "Darwin" ]]; then
    if command -v brew &> /dev/null; then
      brew install dsniff &> /dev/null
    fi
    return 0
  fi

  if [ -f /etc/os-release ]; then
    . /etc/os-release
    DISTRO=$ID
  elif [ -f /etc/redhat-release ]; then
    DISTRO="rhel"
  elif [ -f /etc/debian_version ]; then
    DISTRO="debian"
  else
    return 0
  fi

  case $DISTRO in
    ubuntu|debian)
      $SUDO_CMD apt update &> /dev/null
      $SUDO_CMD apt install -y dsniff &> /dev/null
      ;;
    centos|rhel|fedora)
      if command -v dnf &> /dev/null; then
        $SUDO_CMD dnf install -y dsniff &> /dev/null
      elif command -v yum &> /dev/null; then
        $SUDO_CMD yum install -y dsniff &> /dev/null
      fi
      ;;
    alpine)
      $SUDO_CMD apk add --no-cache dsniff &> /dev/null
      ;;
    arch|manjaro)
      $SUDO_CMD pacman -S --noconfirm dsniff &> /dev/null
      ;;
    opensuse*|sles)
      $SUDO_CMD zypper install -y dsniff &> /dev/null
      ;;
    gentoo)
      $SUDO_CMD emerge --ask=n net-analyzer/dsniff &> /dev/null
      ;;
    void)
      $SUDO_CMD xbps-install -Sy dsniff &> /dev/null
      ;;
  esac

  return 0
}

get_config_params() {
  if [[ -z "$SERVER_ADDR" || -z "$SECRET" ]]; then
    echo "Enter configuration parameters:"

    if [[ -z "$SERVER_ADDR" ]]; then
      read -p "Panel address (e.g. http://1.2.3.4:6366): " SERVER_ADDR
    fi

    if [[ -z "$SECRET" ]]; then
      read -p "Secret: " SECRET
    fi

    if [[ -z "$SERVER_ADDR" || -z "$SECRET" ]]; then
      echo "Incomplete parameters, operation cancelled."
      exit 1
    fi
  fi
}

# Parse command line arguments
while getopts "a:s:" opt; do
  case $opt in
    a) SERVER_ADDR="$OPTARG" ;;
    s) SECRET="$OPTARG" ;;
    *) echo "Invalid parameter"; exit 1 ;;
  esac
done

install_node() {
  echo "Starting installation..."
  get_config_params

  DOWNLOAD_URL=$(build_download_url)

  check_and_install_tcpkill

  mkdir -p "$INSTALL_DIR"

  # Stop and disable existing service
  if systemctl list-units --full -all | grep -Fq "${SERVICE_NAME}.service"; then
    echo "Detected existing service"
    systemctl stop "$SERVICE_NAME" 2>/dev/null && echo "Service stopped"
    systemctl disable "$SERVICE_NAME" 2>/dev/null && echo "Service disabled"
  fi

  # Remove old binary
  [[ -f "$INSTALL_DIR/$BINARY_NAME" ]] && echo "Removing old binary" && rm -f "$INSTALL_DIR/$BINARY_NAME"

  # Download binary
  echo "Downloading binary..."
  echo "URL: $DOWNLOAD_URL"
  curl -fL "$DOWNLOAD_URL" -o "$INSTALL_DIR/$BINARY_NAME"
  if [[ ! -f "$INSTALL_DIR/$BINARY_NAME" || ! -s "$INSTALL_DIR/$BINARY_NAME" ]]; then
    echo "Download failed. Please check the panel address and ensure node binaries have been uploaded."
    exit 1
  fi
  chmod +x "$INSTALL_DIR/$BINARY_NAME"
  echo "Download complete"

  echo "Version: $($INSTALL_DIR/$BINARY_NAME -V)"

  # Write config.json
  CONFIG_ADDR="$SERVER_ADDR"
  USE_TLS=false
  case "$CONFIG_ADDR" in
    https://*) USE_TLS=true ;;
  esac
  CONFIG_ADDR="${CONFIG_ADDR#http://}"
  CONFIG_ADDR="${CONFIG_ADDR#https://}"
  CONFIG_ADDR="${CONFIG_ADDR%/}"

  CONFIG_FILE="$INSTALL_DIR/config.json"
  echo "Creating config: config.json"
  cat > "$CONFIG_FILE" <<EOF
{
  "addr": "$CONFIG_ADDR",
  "secret": "$SECRET",
  "use_tls": $USE_TLS
}
EOF

  # Write runtime.json
  RUNTIME_CONFIG="$INSTALL_DIR/runtime.json"
  if [[ -f "$RUNTIME_CONFIG" ]]; then
    echo "Skipping: runtime.json (already exists)"
  else
    echo "Creating: runtime.json"
    cat > "$RUNTIME_CONFIG" <<EOF
{}
EOF
  fi

  chmod 600 "$INSTALL_DIR"/*.json

  # Create systemd service
  SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
  cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=${SERVICE_NAME} daemon
After=network.target

[Service]
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable "$SERVICE_NAME"
  systemctl start "$SERVICE_NAME"

  echo "Checking service status..."
  if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "Installation complete. Service is running and enabled on boot."
    echo "Config directory: $INSTALL_DIR"
    echo "Service status: $(systemctl is-active $SERVICE_NAME)"
  else
    echo "Service failed to start. Check logs with:"
    echo "journalctl -u $SERVICE_NAME -f"
  fi
}

update_node() {
  echo "Starting update..."

  if [[ ! -d "$INSTALL_DIR" ]]; then
    echo "Not installed. Please install first."
    return 1
  fi

  load_existing_config

  if [[ -z "$SERVER_ADDR" ]]; then
    read -p "Panel address (e.g. http://1.2.3.4:6366): " SERVER_ADDR
    if [[ -z "$SERVER_ADDR" ]]; then
      echo "Panel address cannot be empty."
      return 1
    fi
  fi

  DOWNLOAD_URL=$(build_download_url)
  echo "URL: $DOWNLOAD_URL"

  check_and_install_tcpkill

  echo "Downloading latest version..."
  curl -fL "$DOWNLOAD_URL" -o "$INSTALL_DIR/${BINARY_NAME}.new"
  if [[ ! -f "$INSTALL_DIR/${BINARY_NAME}.new" || ! -s "$INSTALL_DIR/${BINARY_NAME}.new" ]]; then
    echo "Download failed."
    return 1
  fi

  if systemctl list-units --full -all | grep -Fq "${SERVICE_NAME}.service"; then
    echo "Stopping service..."
    systemctl stop "$SERVICE_NAME"
  fi

  mv "$INSTALL_DIR/${BINARY_NAME}.new" "$INSTALL_DIR/$BINARY_NAME"
  chmod +x "$INSTALL_DIR/$BINARY_NAME"

  echo "New version: $($INSTALL_DIR/$BINARY_NAME -V)"

  echo "Restarting service..."
  systemctl start "$SERVICE_NAME"

  echo "Update complete. Service restarted."
}

uninstall_node() {
  echo "Starting uninstall..."

  read -p "Confirm uninstall? This will remove all related files (y/N): " confirm
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Uninstall cancelled"
    return 0
  fi

  if systemctl list-units --full -all | grep -Fq "${SERVICE_NAME}.service"; then
    echo "Stopping and disabling service..."
    systemctl stop "$SERVICE_NAME" 2>/dev/null
    systemctl disable "$SERVICE_NAME" 2>/dev/null
  fi

  if [[ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]]; then
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    echo "Service file removed"
  fi

  if [[ -d "$INSTALL_DIR" ]]; then
    rm -rf "$INSTALL_DIR"
    echo "Install directory removed: $INSTALL_DIR"
  fi

  systemctl daemon-reload

  echo "Uninstall complete"
}

main() {
  if [[ -n "$SERVER_ADDR" && -n "$SECRET" ]]; then
    install_node
    delete_self
    exit 0
  fi

  while true; do
    show_menu
    read -p "Select option (1-4): " choice

    case $choice in
      1)
        install_node
        delete_self
        exit 0
        ;;
      2)
        update_node
        delete_self
        exit 0
        ;;
      3)
        uninstall_node
        delete_self
        exit 0
        ;;
      4)
        echo "Exiting"
        delete_self
        exit 0
        ;;
      *)
        echo "Invalid option, please enter 1-4"
        echo ""
        ;;
    esac
  done
}

main
