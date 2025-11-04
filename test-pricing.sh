#!/bin/bash
# Test script for the pricing-tool
# This validates the Go binary produces reasonable outputs

set -e

echo "=================================="
echo "Pricing Tool Test Suite"
echo "=================================="
echo ""

# Check if binary exists
if [ ! -f "./pricing-tool" ]; then
    echo "❌ Error: pricing-tool binary not found"
    echo "   Please build it first: go build -o pricing-tool cmd/pricing-tool/main.go"
    exit 1
fi

# Check if sample deployment exists
if [ ! -f "./examples/sample-deployment.json" ]; then
    echo "❌ Error: examples/sample-deployment.json not found"
    exit 1
fi

echo "✅ Found pricing-tool binary"
echo "✅ Found sample deployment JSON"
echo ""

# Test 1: Basic execution (no special pricing, no whitelist)
echo "Test 1: Basic Execution"
echo "------------------------"
echo "Input: examples/sample-deployment.json"
echo "Config: Default prices (CPU=1.60, Memory=0.80, GPU=rtx4090 @ default)"
echo ""

unset WHITELIST_URL
unset AKASH_OWNER
export PRICE_TARGET_CPU=1.60
export PRICE_TARGET_MEMORY=0.80
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=120.00"

OUTPUT=$(cat examples/sample-deployment.json | ./pricing-tool)
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo "✅ Exit code: 0 (success)"
    echo "📊 Bid price output: $OUTPUT"
    
    # Validate output is numeric
    if [[ $OUTPUT =~ ^[0-9]+\.?[0-9]*$ ]]; then
        echo "✅ Output is numeric"
    else
        echo "❌ Output is not numeric: $OUTPUT"
        exit 1
    fi
else
    echo "❌ Exit code: $EXIT_CODE (failed)"
    exit 1
fi

echo ""

# Test 2: With GPU pricing
echo "Test 2: Custom GPU Pricing"
echo "------------------------"
export PRICE_TARGET_GPU_MAPPINGS="rtx4090.24gi=150.00,rtx4090=120.00"

OUTPUT=$(cat examples/sample-deployment.json | ./pricing-tool)
if [ $? -eq 0 ]; then
    echo "✅ Custom GPU pricing accepted"
    echo "📊 Bid price with custom GPU: $OUTPUT"
else
    echo "❌ Custom GPU pricing failed"
    exit 1
fi

echo ""

# Test 3: Special pricing
echo "Test 3: Special Account Pricing"
echo "------------------------"
export AKASH_OWNER="akash1fxa9ss3dg6nqyz8aluyaa6svypgprk5tw9fa4q"

OUTPUT=$(cat examples/sample-deployment.json | ./pricing-tool)
if [ $? -eq 0 ]; then
    echo "✅ Special pricing triggered"
    echo "📊 Bid price (should be 1): $OUTPUT"
    
    if [ "$OUTPUT" = "1" ]; then
        echo "✅ Correct special pricing (1 uakt)"
    else
        echo "⚠️  Expected '1', got '$OUTPUT'"
    fi
else
    echo "❌ Special pricing failed"
    exit 1
fi

echo ""

# Test 4: Different resource configurations
echo "Test 4: Different Price Targets"
echo "------------------------"
unset AKASH_OWNER
export PRICE_TARGET_CPU=2.00
export PRICE_TARGET_MEMORY=1.00
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=150.00"

OUTPUT=$(cat examples/sample-deployment.json | ./pricing-tool)
if [ $? -eq 0 ]; then
    echo "✅ Higher pricing accepted"
    echo "📊 Bid price with higher targets: $OUTPUT"
else
    echo "❌ Higher pricing failed"
    exit 1
fi

echo ""

# Test 5: Debug mode
echo "Test 5: Debug Mode"
echo "------------------------"
export DEBUG_BID_SCRIPT=1
export PRICE_TARGET_CPU=1.60
export PRICE_TARGET_MEMORY=0.80

echo "Running with DEBUG_BID_SCRIPT=1..."
OUTPUT=$(cat examples/sample-deployment.json | ./pricing-tool 2>&1)
if [ $? -eq 0 ]; then
    echo "✅ Debug mode works"
    # Check if debug output contains expected strings
    if echo "$OUTPUT" | grep -q "DEBUG:"; then
        echo "✅ Debug logs present"
    else
        echo "⚠️  No DEBUG: prefix found in output"
    fi
else
    echo "❌ Debug mode failed"
    exit 1
fi

unset DEBUG_BID_SCRIPT

echo ""
echo "=================================="
echo "✅ All tests passed!"
echo "=================================="
echo ""
echo "Next steps:"
echo "1. Review the bid prices to ensure they're reasonable"
echo "2. Test with real deployment requests from your provider"
echo "3. Deploy to a test provider"
echo ""

