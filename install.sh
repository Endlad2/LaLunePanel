#!/bin/bash
# LaLune Panel one-click installer.
# Usage: curl -fsSL https://raw.githubusercontent.com/Endlad2/LaLunePanel/master/install.sh | bash
#
# Handles both interactive (tty) and piped (curl | bash) execution.

set -e

REPO_URL="https://github.com/Endlad2/LaLunePanel.git"
BRANCH="master"
DEFAULT_PORT=6333
IMAGE_NAME="lalune-panel"
CONTAINER_NAME="lalune-panel"

# --- stdin handling for curl | bash -----------------------------------------
# When piped, bash reads the script from fd 0. Redirecting fd 0 to /dev/tty
# would cut bash off from its own source mid-script. So if stdin isn't a tty,
# fetch a real copy and re-exec it with stdin on the terminal.
if [ ! -t 0 ]; then
    if [ ! -r /dev/tty ]; then
        echo "[X] No interactive terminal (stdin is not a tty, /dev/tty unreadable)." >&2
        exit 1
    fi
    tmp=$(mktemp -t lalune-install.XXXXXX)
    curl -fsSL "https://raw.githubusercontent.com/Endlad2/LaLunePanel/$BRANCH/install.sh" -o "$tmp"
    chmod +x "$tmp"
    exec bash "$tmp" "$@" < /dev/tty
fi

echo "==============================="
echo "   LaLune Panel Installer"
echo "==============================="
echo ""

# --- privilege helper -------------------------------------------------------
SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
        SUDO="sudo"
    elif command -v doas >/dev/null 2>&1; then
        SUDO="doas"
    else
        echo "[X] Not root and no sudo/doas. Run as root."
        exit 1
    fi
fi

# --- package install --------------------------------------------------------
install_pkg() {
    local pkg="$1"
    echo "[!] Installing $pkg..."
    if command -v apt >/dev/null 2>&1; then
        $SUDO apt update && $SUDO apt install -y "$pkg"
    elif command -v dnf >/dev/null 2>&1; then
        $SUDO dnf install -y "$pkg"
    elif command -v yum >/dev/null 2>&1; then
        $SUDO yum install -y "$pkg"
    elif command -v pacman >/dev/null 2>&1; then
        $SUDO pacman -Sy --noconfirm "$pkg"
    elif command -v apk >/dev/null 2>&1; then
        $SUDO apk add --no-cache "$pkg"
    else
        echo "[X] Unsupported package manager. Install $pkg manually."
        exit 1
    fi
}

command -v git    >/dev/null 2>&1 || install_pkg git
command -v docker >/dev/null 2>&1 || install_pkg docker.io

# docker compose is either a plugin or a standalone binary
if docker compose version >/dev/null 2>&1; then
    COMPOSE="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE="docker-compose"
else
    echo "[!] docker compose not found, installing..."
    install_pkg docker-compose-plugin 2>/dev/null || install_pkg docker-compose
    if docker compose version >/dev/null 2>&1; then
        COMPOSE="docker compose"
    else
        COMPOSE="docker-compose"
    fi
fi

echo "[+] git, docker, compose available"
echo ""

# --- action menu ------------------------------------------------------------
echo "Select action:"
echo "  1) install"
echo "  2) uninstall"
read -p "Enter choice [1-2, default: 1]: " ACTION
ACTION=${ACTION:-1}

# --- uninstall --------------------------------------------------------------
if [ "$ACTION" = "2" ]; then
    echo ""
    echo "[*] Uninstalling LaLune Panel..."

    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
        docker rm   "$CONTAINER_NAME" >/dev/null 2>&1 || true
        echo "[+] Container removed"
    else
        echo "[*] No container named $CONTAINER_NAME"
    fi

    if docker images --format '{{.Repository}}' | grep -q "^${IMAGE_NAME}$"; then
        docker rmi "$IMAGE_NAME" >/dev/null 2>&1 || true
        echo "[+] Image removed"
    fi

    read -p "Remove panel data (clients, configs)? (y/N): " PURGE
    if [[ "$PURGE" =~ ^[Yy]$ ]]; then
        $SUDO rm -rf /opt/lalune
        echo "[+] Data removed (/opt/lalune)"
    fi

    echo ""
    echo "[+] Uninstall complete."
    exit 0
fi

# --- install ----------------------------------------------------------------
echo ""
read -p "Panel port [default: $DEFAULT_PORT]: " PORT_INPUT
PORT=${PORT_INPUT:-$DEFAULT_PORT}

if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
    echo "[X] Invalid port: $PORT"
    exit 1
fi

echo "[*] Panel will listen on port $PORT"
echo ""

INSTALL_DIR="/opt/lalune"

echo "[*] Cloning repository..."
$SUDO rm -rf "$INSTALL_DIR"
$SUDO git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"

cd "$INSTALL_DIR"

echo "[*] Building image (this may take a few minutes)..."
$SUDO docker build -t "$IMAGE_NAME" .

# stop any old instance
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    $SUDO docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    $SUDO docker rm   "$CONTAINER_NAME" >/dev/null 2>&1 || true
fi

echo "[*] Starting container..."
$SUDO docker run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "${PORT}:6333" \
    -v /opt/lalune/data:/data \
    "$IMAGE_NAME"

sleep 2

if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo ""
    echo "==============================="
    echo "[+] LaLune Panel is running!"
    echo "==============================="
    echo ""
    echo "  URL:  http://$(hostname -I 2>/dev/null | awk '{print $1}'):${PORT}"
    echo "  Port: ${PORT}"
    echo ""
    echo "  Logs:  docker logs -f $CONTAINER_NAME"
    echo "  Stop:  docker stop $CONTAINER_NAME"
    echo ""
else
    echo "[X] Container failed to start. Check logs:"
    echo "    docker logs $CONTAINER_NAME"
    exit 1
fi
