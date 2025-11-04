# Production Validation Guide

This guide explains how to validate the Go pricing tool produces the same results as your current bash script in production.

## Strategy

1. **Capture baseline bids** from current provider (bash script)
2. **Extract deployment specs** from logs
3. **Test with Go tool** using same specs
4. **Compare prices** - should match within 1-5%
5. **Deploy Go tool** once validated

## Step 1: Capture Baseline Bids

On your provider, capture some bid logs:

```bash
# Get recent bids
kubectl logs akash-provider-0 -n akash-services | grep "submitting fulfillment" | tail -20

# Example output:
# 5:14PM DBG submitting fulfillment module=bidengine-order order=akash1wd9h4t4cuq6xcj5zkmdqkzq4y9zvh6s72t4z7a/24042650/1/1 price=1.850560000000000000uakt
```

For each bid, also capture the full order context:

```bash
# Get full order details (use the order ID from above)
kubectl logs akash-provider-0 -n akash-services | grep "24042650"
```

Look for the line with `reservation requested` - it contains the full resource spec.

## Step 2: Parse Resources

From the logs, extract:

```
resources=[{"resource":{"id":1,"cpu":{"units":{"val":"100"}},"memory":{"size":{"val":"268435456"}},"storage":[{"name":"default","size":{"val":"268435456"}}],"gpu":{"units":{"val":"0"}},"endpoints":[{"sequence_number":0}]},"count":1,"price":{"denom":"uakt","amount":"100000.000000000000000000"}}]
```

Parse out:
- **CPU:** `"val":"100"` = 100 millicores
- **Memory:** `"val":"268435456"` = 268,435,456 bytes
- **Storage:** `"val":"268435456"` = 268,435,456 bytes
- **GPU:** `"val":"0"` = none
- **Price:** `"amount":"100000..."` = max price
- **Your bid:** `price=1.850560...uakt` = your actual bid

## Step 3: Create Test JSON

Create a JSON file matching the deployment:

```json
{
  "resources": [
    {
      "memory": 268435456,
      "cpu": 100,
      "storage": [{"class": "default", "size": 268435456}],
      "count": 1,
      "endpoint_quantity": 1,
      "ip_lease_quantity": 0
    }
  ],
  "price": {
    "denom": "uakt",
    "amount": "100000.000000000000000000"
  },
  "price_precision": 18
}
```

Save as `examples/real-provider-test.json`

## Step 4: Test with Go Tool

```bash
# Make sure AKT price cache matches your provider's current AKT price
echo "2.50" > /tmp/aktprice.cache

# Test
cat examples/real-provider-test.json | ./pricing-tool
```

## Step 5: Compare Results

Use the comparison script:

```bash
chmod +x compare-with-baseline.sh
./compare-with-baseline.sh
```

Expected output:
```
================================================
Provider Pricing Comparison Tool
================================================

💰 AKT Price: $2.50 (from cache)

📋 BASELINE (from provider logs):
   Bid Price: 1.850560 uakt/block

🧪 TESTING Go Pricing Tool:
   ✅ Go Tool Output: 1.850560 uakt/block

================================================
COMPARISON
================================================

Bash Script:         1.850560 uakt/block
Go Tool:             1.850560 uakt/block

Difference: 0.000000 uakt/block (0.00%)

✅ EXCELLENT! Prices match within 1%
```

## Acceptable Differences

### Why Prices Might Differ Slightly

**Expected differences (< 1%):**
- Floating point rounding
- AKT price fetch timing (different second = different price)
- Precision differences (18 vs 6 decimal places)

**Concerning differences (> 5%):**
- Configuration mismatch (different PRICE_TARGET_* values)
- Different AKT price used
- Bug in conversion logic

## Validation Checklist

Before deploying to production:

- [ ] Captured at least 3-5 baseline bids
- [ ] Created matching JSON test files
- [ ] All comparisons within 5% difference
- [ ] Tested with GPU deployments (if you have GPUs)
- [ ] Tested with different storage types
- [ ] Verified environment variables match provider config
- [ ] Confirmed AKT price source is consistent

## Common Scenarios to Test

### Scenario 1: CPU-Only (Small)
```
CPU: 100 millicores (0.1 cores)
Memory: 256 MB
Storage: 256 MB ephemeral
Expected: ~1-2 uakt/block
```

### Scenario 2: CPU-Only (Large)
```
CPU: 4000 millicores (4 cores)
Memory: 8 GB
Storage: 100 GB ephemeral
Expected: ~20-30 uakt/block
```

### Scenario 3: GPU Deployment
```
CPU: 2000 millicores (2 cores)
Memory: 8 GB
GPU: 1x RTX 4090
Storage: 50 GB ephemeral
Expected: ~300-500 uakt/block
```

### Scenario 4: Multi-GPU High-End
```
CPU: 16000 millicores (16 cores)
Memory: 128 GB
GPU: 4x A100 80GB
Storage: 1 TB NVMe
Expected: ~2000-3000 uakt/block
```

## Debugging Price Mismatches

If Go tool produces very different prices:

### Check Environment Variables

On your provider:
```bash
kubectl exec -it akash-provider-0 -n akash-services -- env | grep PRICE_TARGET
```

In your test:
```bash
export PRICE_TARGET_CPU=1.60
export PRICE_TARGET_MEMORY=0.80
export PRICE_TARGET_HD_EPHEMERAL=0.02
# ... etc
cat examples/real-provider-test.json | ./pricing-tool
```

### Check AKT Price

Provider:
```bash
kubectl exec -it akash-provider-0 -n akash-services -- cat /tmp/aktprice.cache
```

Your test:
```bash
cat /tmp/aktprice.cache
```

### Enable Debug Mode

```bash
export DEBUG_BID_SCRIPT=1
cat examples/real-provider-test.json | ./pricing-tool
```

Look for the breakdown:
```
DEBUG: Resources requested:
DEBUG:   CPU: 0.10 cores
DEBUG:   Memory: 0.25 GB
DEBUG: Pricing breakdown:
DEBUG:   CPU: 0.10 * $1.60 = $0.16
DEBUG:   Memory: 0.25 * $0.80 = $0.20
DEBUG:   Total: $0.41/month
```

## Real Provider Test Example

Your actual baseline:
```
Order: akash1wd9h4t4cuq6xcj5zkmdqkzq4y9zvh6s72t4z7a/24042650/1/1
Resources:
  - CPU: 0.1 cores
  - Memory: 256 MB
  - Storage: 256 MB (ephemeral)
  - Endpoints: 1
Bid: 1.850560 uakt/block
```

Test file created: `examples/real-provider-test.json`

To validate:
```bash
cat examples/real-provider-test.json | ./pricing-tool
# Should output: 1.850560 (or very close)
```

## Next Steps After Validation

Once prices match (within 5%):

1. **Build for Linux:** `GOOS=linux GOARCH=amd64 go build -o pricing-tool cmd/pricing-tool/main.go`
2. **Deploy to test provider** (not production yet!)
3. **Monitor bids for 24-48 hours**
4. **Compare win rates** with bash version
5. **Deploy to production** if stable

## Monitoring After Deployment

After deploying Go tool to provider:

```bash
# Check recent bids
kubectl logs akash-provider-0 -n akash-services | grep "submitting fulfillment" | tail -20

# Enable debug logging
kubectl set env deployment/akash-provider DEBUG_BID_SCRIPT=1 -n akash-services

# Check debug logs
kubectl logs akash-provider-0 -n akash-services | grep "DEBUG:"
```

