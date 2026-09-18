#!/bin/bash
# LaLune Panel one-click installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Endlad2/LaLunePanel/master/install.sh | bash
#
#   ./install.sh                 # pull prebuilt image from GHCR (fast, recommended)
#   ./install.sh --build         # build image locally from source
#   ./install.sh --tag=v1.2.3    # use a specific image tag
#   ./install.sh --port=8080     # skip the port prompt
#
# Handles both interactive (tty) and piped (curl | bash) execution.

set -e

# ---------------------------------------------------------------------------
# config
# ---------------------------------------------------------------------------
REPO_URL="https://github.com/Endlad2/LaLunePanel.git"
BRANCH="master"
DEFAULT_PORT=6333
IMAGE_GHCR="ghcr.io/endlad2/lalunepanel"
IMAGE_LOCAL="lalune-panel"
CONTAINER_NAME="lalune-panel"
INSTALL_DIR="/opt/lalune"

USE_BUILD=0
IMAGE_TAG="latest"
PORT_OVERRIDE=""

# ---------------------------------------------------------------------------
# stdin handling for curl | bash
#
# When piped, bash reads the script from fd 0. Redirecting fd 0 to /dev/tty
# would cut bash off from its own source mid-script. So if stdin isn't a tty,
# fetch a real copy and re-exec it with stdin on the terminal.
# ---------------------------------------------------------------------------
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

# ---------------------------------------------------------------------------
# parse args
# ---------------------------------------------------------------------------
for arg in "$@"; do
    case "$arg" in
        --build)     USE_BUILD=1 ;;
        --tag=*)     IMAGE_TAG="${arg#*=}" ;;
        --port=*)    PORT_OVERRIDE="${arg#*=}" ;;
        --branch=*)  BRANCH="${arg#*=}" ;;
        *) ;;
    esac
done

echo "==============================="
echo "   LaLune Panel Installer"
echo "==============================="
echo ""

# ---------------------------------------------------------------------------
# privilege helper
# ---------------------------------------------------------------------------
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

# ---------------------------------------------------------------------------
# package install
# ---------------------------------------------------------------------------
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

command -v curl >/dev/null 2>&1 || install_pkg curl
command -v docker >/dev/null 2>&1 || install_pkg docker.io

# ---------------------------------------------------------------------------
# action menu
# ---------------------------------------------------------------------------
echo "Select action:"
echo "  1) install"
echo "  2) uninstall"
read -p "Enter choice [1-2, default: 1]: " ACTION
ACTION=${ACTION:-1}

# ---------------------------------------------------------------------------
# uninstall
# ---------------------------------------------------------------------------
if [ "$ACTION" = "2" ]; then
    echo ""
    echo "[*] Uninstalling LaLune Panel..."

    if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        $SUDO docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
        $SUDO docker rm   "$CONTAINER_NAME" >/dev/null 2>&1 || true
        echo "[+] Container removed"
    else
        echo "[*] No container named $CONTAINER_NAME"
    fi

    for img in "$IMAGE_LOCAL" "$IMAGE_GHCR"; do
        if docker images --format '{{.Repository}}' | grep -q "^${img}$"; then
            $SUDO docker rmi "$img" >/dev/null 2>&1 || true
            echo "[+] Image removed: $img"
        fi
    done

    read -p "Remove panel data (clients, configs)? (y/N): " PURGE
    if [[ "$PURGE" =~ ^[Yy]$ ]]; then
        $SUDO rm -rf "$INSTALL_DIR"
        echo "[+] Data removed ($INSTALL_DIR)"
    fi

    echo ""
    echo "[+] Uninstall complete."
    exit 0
fi

# ---------------------------------------------------------------------------
# install
# ---------------------------------------------------------------------------
if [ -n "$PORT_OVERRIDE" ]; then
    PORT="$PORT_OVERRIDE"
else
    read -p "Panel port [default: $DEFAULT_PORT]: " PORT_INPUT
    PORT=${PORT_INPUT:-$DEFAULT_PORT}
fi

if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
    echo "[X] Invalid port: $PORT"
    exit 1
fi

echo "[*] Panel will listen on port $PORT"
echo ""

# choose image source
if [ "$USE_BUILD" = "1" ]; then
    echo "[*] Mode: local build (--build)"
    IMAGE_REF="$IMAGE_LOCAL:$IMAGE_TAG"
    IMAGE_SOURCE="local"
else
    echo "[*] Mode: pull from GHCR"
    IMAGE_REF="$IMAGE_GHCR:$IMAGE_TAG"
    IMAGE_SOURCE="ghcr"
    echo "[*] Image: $IMAGE_REF"
fi
echo ""

# ---------------------------------------------------------------------------
# acquire the image
# ---------------------------------------------------------------------------
if [ "$IMAGE_SOURCE" = "ghcr" ]; then
    echo "[*] Pulling image from GHCR..."
    if ! $SUDO docker pull "$IMAGE_REF"; then
        echo ""
        echo "[!] Pull failed."
        echo "    Possible causes:"
        echo "      - the GHCR package is private (make it public in repo settings)"
        echo "      - the tag '$IMAGE_TAG' doesn't exist yet"
        echo "      - you're not logged in:  echo \$TOKEN | docker login ghcr.io -u USER --password-stdin"
        echo ""
        read -p "Build locally instead? (Y/n): " FALLBACK
        if [[ ! "$FALLBACK" =~ ^[Nn]$ ]]; then
            USE_BUILD=1
            IMAGE_REF="$IMAGE_LOCAL:$IMAGE_TAG"
            IMAGE_SOURCE="local"
        else
            exit 1
        fi
    fi
fi

if [ "$IMAGE_SOURCE" = "local" ]; then
    command -v git >/dev/null 2>&1 || install_pkg git

    echo "[*] Cloning repository..."
    $SUDO rm -rf "$INSTALL_DIR"
    $SUDO git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$INSTALL_DIR"

    cd "$INSTALL_DIR"

    echo "[*] Building image (this may take a few minutes)..."
    $SUDO docker build -t "$IMAGE_REF" .
fi

# ---------------------------------------------------------------------------
# (re)start container
# ---------------------------------------------------------------------------
mkdir -p "$INSTALL_DIR/data" 2>/dev/null || $SUDO mkdir -p "$INSTALL_DIR/data"

if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    $SUDO docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
    $SUDO docker rm   "$CONTAINER_NAME" >/dev/null 2>&1 || true
fi

echo "[*] Starting container..."
$SUDO docker run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "${PORT}:6333" \
    -v "$INSTALL_DIR/data:/data" \
    "$IMAGE_REF"

sleep 2

if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    IP=$(hostname -I 2>/dev/null | awk '{print $1}')
    [ -z "$IP" ] && IP="localhost"
    echo ""
    echo "==============================="
    echo "[+] LaLune Panel is running!"
    echo "==============================="
    echo ""
    echo "  URL:   http://${IP}:${PORT}"
    echo "  Image: $IMAGE_REF"
    echo "  Port:  ${PORT}"
    echo ""
    echo "  Logs:  docker logs -f $CONTAINER_NAME"
    echo "  Stop:  docker stop $CONTAINER_NAME"
    echo ""
else
    echo "[X] Container failed to start. Check logs:"
    echo "    docker logs $CONTAINER_NAME"
    exit 1
fi
