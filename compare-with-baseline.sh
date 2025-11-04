#!/bin/bash
# Compare Go pricing tool output with baseline provider bids

echo "================================================"
echo "Provider Pricing Comparison Tool"
echo "================================================"
echo ""

# Check if pricing-tool exists
if [ ! -f "./pricing-tool" ]; then
    echo "❌ Error: pricing-tool binary not found"
    echo "   Build it first: go build -o pricing-tool cmd/pricing-tool/main.go"
    exit 1
fi

# Ensure AKT price is cached
if [ ! -f "/tmp/aktprice.cache" ]; then
    echo "⚠️  Warning: No AKT price cache found"
    echo "   Creating cache with price: $2.50"
    echo "2.50" > /tmp/aktprice.cache
fi

AKT_PRICE=$(cat /tmp/aktprice.cache)
echo "💰 AKT Price: \$$AKT_PRICE (from cache)"
echo ""

# Baseline from your provider logs
echo "📋 BASELINE (from provider logs):"
echo "   Order: akash1wd9h4t4cuq6xcj5zkmdqkzq4y9zvh6s72t4z7a/24042650/1/1"
echo "   Resources:"
echo "     - CPU: 100 millicores (0.1 cores)"
echo "     - Memory: 268435456 bytes (256 MB)"
echo "     - Storage: 268435456 bytes (256 MB, ephemeral)"
echo "     - GPU: None"
echo "     - Endpoints: 1"
echo "   Bid Price: 1.850560 uakt/block"
echo ""

# Test with Go pricing tool
echo "🧪 TESTING Go Pricing Tool:"
echo "   File: examples/real-provider-test.json"
echo ""

# Set default pricing (adjust these to match your current provider config)
export PRICE_TARGET_CPU=1.60
export PRICE_TARGET_MEMORY=0.80
export PRICE_TARGET_HD_EPHEMERAL=0.02
export PRICE_TARGET_ENDPOINT=0.05

GO_OUTPUT=$(cat examples/real-provider-test.json | ./pricing-tool 2>&1)
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    # Extract numeric price (last line)
    GO_PRICE=$(echo "$GO_OUTPUT" | tail -1)
    
    echo "   ✅ Go Tool Output: $GO_PRICE uakt/block"
    echo ""
    
    # Compare
    echo "================================================"
    echo "COMPARISON"
    echo "================================================"
    echo ""
    printf "%-20s %s\n" "Bash Script:" "1.850560 uakt/block"
    printf "%-20s %s\n" "Go Tool:" "$GO_PRICE uakt/block"
    echo ""
    
    # Calculate difference
    BASH_PRICE="1.850560"
    
    # Use bc for floating point math if available
    if command -v bc &> /dev/null; then
        DIFF=$(echo "$GO_PRICE - $BASH_PRICE" | bc -l)
        PCT_DIFF=$(echo "scale=2; ($DIFF / $BASH_PRICE) * 100" | bc -l)
        
        echo "Difference: $DIFF uakt/block ($PCT_DIFF%)"
        echo ""
        
        # Evaluate
        ABS_PCT=$(echo "$PCT_DIFF" | tr -d '-')
        if (( $(echo "$ABS_PCT < 1" | bc -l) )); then
            echo "✅ EXCELLENT! Prices match within 1%"
        elif (( $(echo "$ABS_PCT < 5" | bc -l) )); then
            echo "✅ GOOD! Prices match within 5%"
        elif (( $(echo "$ABS_PCT < 10" | bc -l) )); then
            echo "⚠️  ACCEPTABLE: Prices differ by less than 10%"
        else
            echo "❌ WARNING: Prices differ by more than 10%"
            echo "   This might indicate a configuration mismatch"
        fi
    else
        echo "Note: Install 'bc' for automatic comparison calculation"
    fi
    
    echo ""
    echo "================================================"
    echo "NOTES"
    echo "================================================"
    echo ""
    echo "Small differences are expected due to:"
    echo "  • AKT price fetch timing differences"
    echo "  • Floating point precision"
    echo "  • Rounding differences"
    echo ""
    echo "If prices differ significantly, check:"
    echo "  • PRICE_TARGET_* environment variables"
    echo "  • AKT price cache (/tmp/aktprice.cache)"
    echo "  • Bash script configuration on provider"
    echo ""
    
else
    echo "   ❌ Go Tool Failed (exit code: $EXIT_CODE)"
    echo "   Output: $GO_OUTPUT"
    exit 1
fi

