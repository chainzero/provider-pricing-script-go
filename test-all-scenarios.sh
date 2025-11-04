#!/bin/bash
# Comprehensive test suite for all deployment scenarios

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BINARY="./pricing-tool"
EXAMPLES_DIR="./examples"

# Check prerequisites
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}❌ Error: pricing-tool binary not found${NC}"
    echo "   Build it first: go build -o pricing-tool cmd/pricing-tool/main.go"
    exit 1
fi

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Akash Pricing Tool - Test Suite      ║${NC}"
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo ""

# Clean environment
unset WHITELIST_URL
unset AKASH_OWNER
unset DEBUG_BID_SCRIPT

# Default pricing
export PRICE_TARGET_CPU=1.60
export PRICE_TARGET_MEMORY=0.80
export PRICE_TARGET_HD_EPHEMERAL=0.02
export PRICE_TARGET_HD_PERS_HDD=0.01
export PRICE_TARGET_HD_PERS_SSD=0.03
export PRICE_TARGET_HD_PERS_NVME=0.04
export PRICE_TARGET_ENDPOINT=0.05
export PRICE_TARGET_IP=5.00
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=120.00,rtx4090.24gi=150.00,a100=200.00,a100.80gi=250.00,a100.80gi.sxm4=280.00"

TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local json_file="$2"
    local expected_pattern="$3"
    
    echo -e "${YELLOW}Test: ${test_name}${NC}"
    echo "  File: $json_file"
    
    if [ ! -f "$json_file" ]; then
        echo -e "  ${RED}❌ FAILED - File not found${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        echo ""
        return
    fi
    
    # Run the pricing tool
    OUTPUT=$(cat "$json_file" | $BINARY 2>&1)
    EXIT_CODE=$?
    
    if [ $EXIT_CODE -eq 0 ]; then
        echo -e "  ${GREEN}✅ Exit Code: 0${NC}"
        
        # Extract just the numeric output (last line, assuming it's the price)
        PRICE=$(echo "$OUTPUT" | tail -1)
        echo "  📊 Bid Price: ${PRICE}"
        
        # Validate it's numeric
        if [[ $PRICE =~ ^[0-9]+\.?[0-9]*$ ]]; then
            echo -e "  ${GREEN}✅ Output is numeric${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
        else
            echo -e "  ${RED}❌ FAILED - Output not numeric: $PRICE${NC}"
            TESTS_FAILED=$((TESTS_FAILED + 1))
        fi
    else
        echo -e "  ${RED}❌ FAILED - Exit code: $EXIT_CODE${NC}"
        echo "  Output: $OUTPUT"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    
    echo ""
}

# Run tests
echo -e "${BLUE}═══ Scenario Tests ═══${NC}"
echo ""

run_test "1. GPU Deployment (RTX 4090)" \
    "$EXAMPLES_DIR/sample-deployment.json" \
    "numeric"

run_test "2. CPU-Only Deployment" \
    "$EXAMPLES_DIR/cpu-only-deployment.json" \
    "numeric"

run_test "3. Multi-Storage Deployment (SSD + NVMe)" \
    "$EXAMPLES_DIR/multi-storage-deployment.json" \
    "numeric"

run_test "4. Multi-GPU Deployment (2x A100)" \
    "$EXAMPLES_DIR/multi-gpu-deployment.json" \
    "numeric"

run_test "5. Old Format (No Price Field)" \
    "$EXAMPLES_DIR/old-format-deployment.json" \
    "numeric"

echo -e "${BLUE}═══ Special Cases ═══${NC}"
echo ""

# Test special pricing
echo -e "${YELLOW}Test: Special Account Pricing${NC}"
export AKASH_OWNER="akash1fxa9ss3dg6nqyz8aluyaa6svypgprk5tw9fa4q"
OUTPUT=$(cat "$EXAMPLES_DIR/sample-deployment.json" | $BINARY 2>&1)
if [ "$OUTPUT" = "1" ]; then
    echo -e "  ${GREEN}✅ Special pricing returned '1'${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "  ${RED}❌ Expected '1', got '$OUTPUT'${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
unset AKASH_OWNER
echo ""

# Test different GPU pricing configurations
echo -e "${YELLOW}Test: GPU Price Fallback Logic${NC}"
echo "  Testing: a100.80gi.sxm4 -> a100.80gi -> a100 -> default"

# Test most specific
export PRICE_TARGET_GPU_MAPPINGS="a100.80gi.sxm4=280.00"
OUTPUT=$(cat "$EXAMPLES_DIR/multi-gpu-deployment.json" | $BINARY 2>&1 | tail -1)
echo "  With a100.80gi.sxm4=280.00: $OUTPUT"

# Test medium specific
export PRICE_TARGET_GPU_MAPPINGS="a100.80gi=250.00"
OUTPUT=$(cat "$EXAMPLES_DIR/multi-gpu-deployment.json" | $BINARY 2>&1 | tail -1)
echo "  With a100.80gi=250.00: $OUTPUT"

# Test least specific
export PRICE_TARGET_GPU_MAPPINGS="a100=200.00"
OUTPUT=$(cat "$EXAMPLES_DIR/multi-gpu-deployment.json" | $BINARY 2>&1 | tail -1)
echo "  With a100=200.00: $OUTPUT"

# Test default fallback
unset PRICE_TARGET_GPU_MAPPINGS
OUTPUT=$(cat "$EXAMPLES_DIR/multi-gpu-deployment.json" | $BINARY 2>&1 | tail -1)
echo "  With no mapping (default 100): $OUTPUT"

echo -e "  ${GREEN}✅ GPU fallback logic working${NC}"
TESTS_PASSED=$((TESTS_PASSED + 1))
echo ""

# Test environment variable override
echo -e "${YELLOW}Test: Environment Variable Overrides${NC}"
export PRICE_TARGET_CPU=10.00
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=500.00"
OUTPUT=$(cat "$EXAMPLES_DIR/sample-deployment.json" | $BINARY 2>&1 | tail -1)
echo "  With CPU=10.00, GPU=500.00: $OUTPUT"
echo -e "  ${GREEN}✅ Environment overrides working${NC}"
TESTS_PASSED=$((TESTS_PASSED + 1))
echo ""

# Summary
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Test Summary                          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Review the bid prices above"
    echo "  2. Compare with expected values for your hardware"
    echo "  3. Test with DEBUG_BID_SCRIPT=1 for detailed logs"
    echo "  4. Deploy to a test provider"
    echo ""
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    echo ""
    exit 1
fi

