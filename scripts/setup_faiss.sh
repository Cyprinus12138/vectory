#!/usr/bin/env bash
set -euo pipefail

# setup_faiss.sh - Build and install FAISS C API library (libfaiss_c.so).
# This is required for building and running Vectory with FAISS index support.
#
# Usage:
#   sudo ./scripts/setup_faiss.sh              # Clone, build and install
#   sudo ./scripts/setup_faiss.sh /path/to/dir # Build in a specific directory
#
# Prerequisites: cmake (>=3.24), g++, git, make

FAISS_REPO="https://github.com/facebookresearch/faiss.git"
INSTALL_PREFIX="/usr/local"

# --- Prerequisite checks ---

check_cmd() {
    if ! command -v "$1" &>/dev/null; then
        echo "Error: '$1' is required but not found. Please install it first."
        exit 1
    fi
}

check_cmd cmake
check_cmd g++
check_cmd git
check_cmd make

echo "All prerequisites satisfied."

# --- Determine build directory ---

BUILD_DIR="${1:-$(mktemp -d /tmp/faiss-build-XXXXXX)}"
mkdir -p "$BUILD_DIR"
echo "Build directory: $BUILD_DIR"

# --- Clone and build ---

if [ ! -d "$BUILD_DIR/faiss/.git" ]; then
    echo "Cloning FAISS..."
    git clone --depth 1 "$FAISS_REPO" "$BUILD_DIR/faiss"
else
    echo "FAISS source already exists, skipping clone."
fi

cd "$BUILD_DIR/faiss"

echo "Configuring FAISS (GPU disabled, C API enabled, shared libs)..."
cmake -B build \
    -DFAISS_ENABLE_GPU=OFF \
    -DFAISS_ENABLE_C_API=ON \
    -DBUILD_SHARED_LIBS=ON \
    -DCMAKE_INSTALL_PREFIX="$INSTALL_PREFIX" \
    -DFAISS_ENABLE_PYTHON=OFF \
    -DBUILD_TESTING=OFF \
    .

echo "Building FAISS (this may take a while)..."
make -C build -j"$(nproc)"

echo "Installing FAISS to $INSTALL_PREFIX..."
make -C build install

# --- Update linker cache ---

ldconfig

# --- Verify ---

if ldconfig -p | grep -q libfaiss_c; then
    echo ""
    echo "FAISS C API installed successfully."
    echo "  libfaiss_c.so: $(ldconfig -p | grep libfaiss_c | awk '{print $NF}')"
else
    echo ""
    echo "Warning: libfaiss_c.so was installed but not found by ldconfig."
    echo "You may need to add $INSTALL_PREFIX/lib to /etc/ld.so.conf.d/ and run ldconfig."
    exit 1
fi

echo ""
echo "You can now build Vectory with: make build"
echo "Build directory retained at: $BUILD_DIR (safe to remove)"
