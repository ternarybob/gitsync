#!/bin/bash
# -----------------------------------------------------------------------
# Test Script for GitSync
# -----------------------------------------------------------------------

set -e

# Default values
PACKAGE="./..."
VERBOSE=false
COVERAGE=false
RACE=false
SHORT=false
BENCH=false
RUN=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --package|-p)
            PACKAGE="$2"
            shift 2
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --coverage|-c)
            COVERAGE=true
            shift
            ;;
        --race)
            RACE=true
            shift
            ;;
        --short|-s)
            SHORT=true
            shift
            ;;
        --bench|-b)
            BENCH=true
            shift
            ;;
        --run|-r)
            RUN="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --package, -p   Specific package to test (default: all)"
            echo "  --verbose, -v   Enable verbose output"
            echo "  --coverage, -c  Generate coverage report"
            echo "  --race          Enable race detector"
            echo "  --short, -s     Run short tests only"
            echo "  --bench, -b     Run benchmarks"
            echo "  --run, -r       Run only tests matching pattern"
            echo "  --help, -h      Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo -e "${CYAN}GitSync Test Runner${NC}"
echo -e "${CYAN}===================${NC}"

# Build test command
TEST_CMD="go test"

if [ "$VERBOSE" = true ]; then
    TEST_CMD="$TEST_CMD -v"
    echo -e "${YELLOW}Verbose: Enabled${NC}"
fi

if [ "$COVERAGE" = true ]; then
    TEST_CMD="$TEST_CMD -cover -coverprofile=coverage.out"
    echo -e "${YELLOW}Coverage: Enabled${NC}"
fi

if [ "$RACE" = true ]; then
    TEST_CMD="$TEST_CMD -race"
    echo -e "${YELLOW}Race Detector: Enabled${NC}"
fi

if [ "$SHORT" = true ]; then
    TEST_CMD="$TEST_CMD -short"
    echo -e "${YELLOW}Short Tests: Enabled${NC}"
fi

if [ "$BENCH" = true ]; then
    TEST_CMD="$TEST_CMD -bench=."
    echo -e "${YELLOW}Benchmarks: Enabled${NC}"
fi

if [ -n "$RUN" ]; then
    TEST_CMD="$TEST_CMD -run $RUN"
    echo -e "${YELLOW}Pattern: $RUN${NC}"
fi

# Add package
TEST_CMD="$TEST_CMD $PACKAGE"
echo -e "${YELLOW}Package: $PACKAGE${NC}"

echo -e "\n${GREEN}Running tests...${NC}"
echo -e "Command: $TEST_CMD"
echo ""

# Run tests
if $TEST_CMD; then
    echo -e "\n${GREEN}✓ All tests passed!${NC}"

    # Show coverage report if generated
    if [ "$COVERAGE" = true ]; then
        echo -e "\n${YELLOW}Generating coverage report...${NC}"
        go tool cover -func=coverage.out

        echo -e "\n${CYAN}To view HTML coverage report, run:${NC}"
        echo "  go tool cover -html=coverage.out"
    fi
else
    echo -e "\n${RED}✗ Tests failed${NC}"
    exit 1
fi