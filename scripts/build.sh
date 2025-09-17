#!/bin/bash
# -----------------------------------------------------------------------
# Build Script for GitSync
# -----------------------------------------------------------------------

set -e

# Default values
ENVIRONMENT="dev"
VERSION=""
CLEAN=false
TEST=false
VERBOSE=false
RELEASE=false
OS=""
ARCH=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --environment|-e)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --version|-v)
            VERSION="$2"
            shift 2
            ;;
        --clean|-c)
            CLEAN=true
            shift
            ;;
        --test|-t)
            TEST=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --release|-r)
            RELEASE=true
            shift
            ;;
        --os)
            OS="$2"
            shift 2
            ;;
        --arch)
            ARCH="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --environment, -e  Target environment (dev, staging, prod)"
            echo "  --version, -v      Version to embed in binary"
            echo "  --clean, -c        Clean build artifacts"
            echo "  --test, -t         Run tests before building"
            echo "  --verbose          Enable verbose output"
            echo "  --release, -r      Build optimized release binary"
            echo "  --os               Target OS (linux, windows, darwin)"
            echo "  --arch             Target architecture (amd64, arm64)"
            echo "  --help, -h         Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo -e "${CYAN}GitSync Build Script${NC}"
echo -e "${YELLOW}Environment: $ENVIRONMENT${NC}"

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(dev|staging|prod)$ ]]; then
    echo -e "${RED}Invalid environment: $ENVIRONMENT${NC}"
    exit 1
fi

# Get version if not provided
if [ -z "$VERSION" ]; then
    # Try to read from .version file first
    if [ -f ".version" ]; then
        VERSION=$(grep "^version:" .version | cut -d' ' -f2 2>/dev/null)
    fi

    # Fall back to git if .version file doesn't exist or version not found
    if [ -z "$VERSION" ]; then
        VERSION=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
    fi
fi
echo -e "${GREEN}Version: $VERSION${NC}"

# Get build time
BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")
echo "Build Time: $BUILD_TIME"

# Clean if requested
if [ "$CLEAN" = true ]; then
    echo -e "\n${YELLOW}Cleaning build artifacts...${NC}"
    rm -rf bin/
    go clean -cache
    echo -e "${GREEN}Clean complete${NC}"
fi

# Run tests if requested
if [ "$TEST" = true ]; then
    echo -e "\n${YELLOW}Running tests...${NC}"
    if [ "$VERBOSE" = true ]; then
        go test -v ./...
    else
        go test ./...
    fi
    echo -e "${GREEN}Tests passed${NC}"
fi

# Create bin directory
mkdir -p bin

# Determine output binary name based on target OS
if [ "$OS" = "windows" ] || [ "$GOOS" = "windows" ] || ([ -z "$OS" ] && [ -z "$GOOS" ] && [ "$OSTYPE" = "msys" ]); then
    OUTPUT_NAME="gitsync.exe"
elif [ "$OS" = "darwin" ] || [ "$GOOS" = "darwin" ] || ([ -z "$OS" ] && [ -z "$GOOS" ] && [ "$(uname)" = "Darwin" ]); then
    OUTPUT_NAME="gitsync-darwin"
else
    OUTPUT_NAME="gitsync-linux"
fi
OUTPUT_PATH="bin/${OUTPUT_NAME}"

# Set up build environment
export CGO_ENABLED=0
if [ -n "$OS" ]; then
    export GOOS=$OS
fi
if [ -n "$ARCH" ]; then
    export GOARCH=$ARCH
fi

# Build flags
LDFLAGS="-X github.com/ternarybob/gitsync/internal/version.Version=$VERSION -X 'github.com/ternarybob/gitsync/internal/version.Build=$BUILD_TIME'"

if [ "$RELEASE" = true ]; then
    echo -e "\n${YELLOW}Building release binary...${NC}"
    LDFLAGS="${LDFLAGS} -s -w"
    BUILD_FLAGS="-trimpath"
else
    echo -e "\n${YELLOW}Building development binary...${NC}"
    BUILD_FLAGS=""
fi

if [ "$VERBOSE" = true ]; then
    BUILD_FLAGS="${BUILD_FLAGS} -v"
    echo "Build command: go build -o $OUTPUT_PATH -ldflags \"$LDFLAGS\" $BUILD_FLAGS ./cmd/gitsync"
fi

# Build the binary
go build -o "$OUTPUT_PATH" -ldflags "$LDFLAGS" $BUILD_FLAGS ./cmd/gitsync

# Display results
echo -e "\n${GREEN}Build successful!${NC}"
echo -e "${YELLOW}Output: $OUTPUT_PATH${NC}"

# Show binary info
SIZE=$(du -h "$OUTPUT_PATH" | cut -f1)
echo "Size: $SIZE"

if [ -n "$OS" ] && [ -n "$ARCH" ]; then
    echo "Target: $OS/$ARCH"
fi

# Create version file
cat > bin/version.txt <<EOF
Version: $VERSION
Build: $BUILD_TIME
Environment: $ENVIRONMENT
EOF

echo -e "\n${GREEN}Build complete!${NC}"