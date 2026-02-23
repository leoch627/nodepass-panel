#!/bin/sh
set -e

CONFIG_FILE="/etc/node/config.json"
RUNTIME_CONFIG="/etc/node/runtime.json"

# Generate config.json from environment variables if set
if [ -n "$PANEL_ADDR" ] && [ -n "$SECRET" ]; then
  echo "Generating config from environment variables..."

  ADDR_VALUE="$PANEL_ADDR"
  USE_TLS=false
  case "$ADDR_VALUE" in
    https://*) USE_TLS=true ;;
  esac
  ADDR_VALUE="${ADDR_VALUE#http://}"
  ADDR_VALUE="${ADDR_VALUE#https://}"
  ADDR_VALUE="${ADDR_VALUE%/}"

  cat > "$CONFIG_FILE" <<EOF
{
  "addr": "$ADDR_VALUE",
  "secret": "$SECRET",
  "use_tls": $USE_TLS,
  "v_bin": "svc-runtime",
  "v_cfg": "service.json"
}
EOF
else
  if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: PANEL_ADDR/SECRET not set and $CONFIG_FILE not found."
    echo "Provide configuration via one of:"
    echo "  1. Environment: -e PANEL_ADDR=http://panel:6366 -e SECRET=<secret>"
    echo "  2. Mount config: -v ./config.json:/etc/node/config.json"
    exit 1
  fi
  echo "Using mounted config: $CONFIG_FILE"
fi

# Ensure runtime config exists
if [ ! -f "$RUNTIME_CONFIG" ]; then
  echo "{}" > "$RUNTIME_CONFIG"
  echo "Created runtime config: $RUNTIME_CONFIG"
else
  echo "Using existing runtime config: $RUNTIME_CONFIG"
fi

# Restore persisted custom node binary
if [ -f /etc/node/node-svc ]; then
  cp /etc/node/node-svc /usr/local/bin/node-svc
  chmod +x /usr/local/bin/node-svc
  echo "Restored persisted node binary"
fi

# Restore persisted secondary binary
if [ -f /etc/node/svc-runtime ]; then
  cp /etc/node/svc-runtime /usr/local/bin/svc-runtime
  chmod +x /usr/local/bin/svc-runtime
  echo "Restored persisted secondary binary"
fi

echo "Starting service..."
exec /usr/local/bin/node-svc
