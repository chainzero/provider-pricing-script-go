# Testing Guide

This guide explains how to test the pricing-tool binary before deploying to production.

## Quick Start

```bash
# 1. Build the binary
go build -o pricing-tool cmd/pricing-tool/main.go

# 2. Run basic tests
chmod +x test-pricing.sh
./test-pricing.sh

# 3. Run comprehensive tests
chmod +x test-all-scenarios.sh
./test-all-scenarios.sh
```

## Test Files

### Test Scripts

- **`test-pricing.sh`** - Basic functionality tests
  - Tests default configuration
  - Tests special pricing
  - Tests debug mode
  - Tests environment variable overrides

- **`test-all-scenarios.sh`** - Comprehensive scenario tests
  - CPU-only deployments
  - GPU deployments (single and multi)
  - Different storage types
  - Old format (backward compatibility)
  - GPU price fallback logic

### Example Deployments

All located in `examples/`:

1. **`sample-deployment.json`** - GPU deployment (RTX 4090)
   - 1 CPU core, 2GB RAM, 10GB storage
   - 1x RTX 4090 GPU with 24GB VRAM

2. **`cpu-only-deployment.json`** - No GPU
   - 2 CPU cores, 4GB RAM, 20GB storage
   - Tests basic resource pricing

3. **`multi-storage-deployment.json`** - Multiple storage types
   - 4 CPU cores, 8GB RAM
   - Ephemeral + SSD (beta2) + NVMe (beta3) storage
   - 2 endpoints, 1 IP lease

4. **`multi-gpu-deployment.json`** - Multiple GPUs
   - 8 CPU cores, 16GB RAM
   - 2x A100 80GB with SXM4 interface
   - Tests GPU fallback logic

5. **`old-format-deployment.json`** - Backward compatibility
   - No `price` field (old format)
   - Should output uakt with no decimal places

## Manual Testing

### Basic Test

```bash
# Test with default configuration
cat examples/sample-deployment.json | ./pricing-tool

# Expected output: A numeric bid price (e.g., "125.450000")
```

### Test with Environment Variables

```bash
# Custom pricing
export PRICE_TARGET_CPU=2.00
export PRICE_TARGET_MEMORY=1.00
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=150.00"

cat examples/sample-deployment.json | ./pricing-tool
```

### Test Special Pricing

```bash
# Alice's account gets special pricing (1 uakt)
export AKASH_OWNER="akash1fxa9ss3dg6nqyz8aluyaa6svypgprk5tw9fa4q"
cat examples/sample-deployment.json | ./pricing-tool

# Expected output: "1"
```

### Debug Mode

```bash
# Enable debug logging
export DEBUG_BID_SCRIPT=1
cat examples/sample-deployment.json | ./pricing-tool

# You'll see detailed logs on stderr:
# DEBUG: Resources requested:
# DEBUG:   CPU: 1.00 cores
# DEBUG:   Memory: 2.000000000000 GB
# DEBUG: Pricing breakdown:
# ...
```

### Test GPU Fallback Logic

```bash
# Most specific: model.vram.interface
export PRICE_TARGET_GPU_MAPPINGS="a100.80gi.sxm4=280.00"
cat examples/multi-gpu-deployment.json | ./pricing-tool

# Medium specific: model.vram
export PRICE_TARGET_GPU_MAPPINGS="a100.80gi=250.00"
cat examples/multi-gpu-deployment.json | ./pricing-tool

# Least specific: model
export PRICE_TARGET_GPU_MAPPINGS="a100=200.00"
cat examples/multi-gpu-deployment.json | ./pricing-tool

# No match: falls back to 100.00
unset PRICE_TARGET_GPU_MAPPINGS
cat examples/multi-gpu-deployment.json | ./pricing-tool
```

## Understanding Output

### Success Case
```
$ cat examples/sample-deployment.json | ./pricing-tool
125.450000
$ echo $?
0
```
- Output is **only** the bid price (to stdout)
- Exit code is `0`

### Error Case
```
$ cat examples/sample-deployment.json | ./pricing-tool
Error getting AKT price: connection timeout
$ echo $?
1
```
- Error message goes to **stderr**
- Exit code is non-zero

### Debug Output
```
$ DEBUG_BID_SCRIPT=1 cat examples/sample-deployment.json | ./pricing-tool
DEBUG: Resources requested:
DEBUG:   CPU: 1.00 cores
DEBUG:   Memory: 2.000000000000 GB
...
125.450000
```
- Debug logs go to **stderr**
- Bid price still goes to **stdout**

## Validating Calculations

### Calculate Expected Price Manually

For `sample-deployment.json`:
- **CPU**: 1 core × $1.60 = $1.60/month
- **Memory**: 2 GB × $0.80 = $1.60/month
- **Storage**: 10 GB × $0.02 = $0.20/month
- **GPU**: 1 × $120.00 = $120.00/month
- **Endpoints**: 1 × $0.05 = $0.05/month
- **Total**: $123.45/month

Convert to per-block rate:
- **Blocks per month**: 429,909
- **AKT price**: ~$2.00 (example)
- **Monthly cost in AKT**: $123.45 / $2.00 = 61.725 AKT
- **Monthly cost in uAKT**: 61,725,000 uAKT
- **Per-block rate**: 61,725,000 / 429,909 ≈ **143.56 uAKT/block**

Compare with tool output!

## Environment Variables Reference

### Pricing Configuration

```bash
export PRICE_TARGET_CPU=1.60              # USD per CPU core/month
export PRICE_TARGET_MEMORY=0.80           # USD per GB RAM/month
export PRICE_TARGET_HD_EPHEMERAL=0.02     # USD per GB ephemeral/month
export PRICE_TARGET_HD_PERS_HDD=0.01      # USD per GB HDD (beta1)/month
export PRICE_TARGET_HD_PERS_SSD=0.03      # USD per GB SSD (beta2)/month
export PRICE_TARGET_HD_PERS_NVME=0.04     # USD per GB NVMe (beta3)/month
export PRICE_TARGET_ENDPOINT=0.05         # USD per endpoint/month
export PRICE_TARGET_IP=5.00               # USD per IP/month
export PRICE_TARGET_GPU_MAPPINGS="model=price,model.vram=price"
```

### Operational Configuration

```bash
export AKASH_OWNER="akash1..."            # Tenant address (set by Provider)
export WHITELIST_URL="https://..."       # URL to whitelist file (optional)
export DEBUG_BID_SCRIPT=1                 # Enable debug logging
```

## Common Issues

### Issue: "Error getting AKT price"

**Cause**: Cannot fetch AKT price from API (network issue or cache miss)

**Solution**:
```bash
# Pre-populate the cache
echo "2.50" > /tmp/aktprice.cache
# Now test
cat examples/sample-deployment.json | ./pricing-tool
```

### Issue: "requested rate is too low"

**Cause**: Your calculated price exceeds the deployment's max price

**Solution**: This is expected behavior! The tool is rejecting an unprofitable bid.

### Issue: Output has extra text

**Cause**: Might have debug output mixed with price

**Solution**:
```bash
# Ensure DEBUG_BID_SCRIPT is not set
unset DEBUG_BID_SCRIPT
cat examples/sample-deployment.json | ./pricing-tool
```

## Next Steps After Testing

1. **Review prices** - Do the bid prices make sense for your hardware?
2. **Test edge cases** - Create deployment JSONs for your specific GPU models
3. **Deploy to test provider** - See if real bids work
4. **Monitor logs** - Check `/tmp/*.log` files with DEBUG_BID_SCRIPT enabled
5. **Compare with bash script** - Ensure prices are similar

## Comparing with Bash Script (Optional)

If you have the original bash script:

```bash
# Go version
cat examples/sample-deployment.json | ./pricing-tool > go-output.txt

# Bash version
cat examples/sample-deployment.json | ./price_script_generic.sh > bash-output.txt

# Compare
diff go-output.txt bash-output.txt
```

Prices might differ slightly due to:
- AKT price fetch timing
- Floating point precision
- Rounding differences

But they should be within ~1% of each other.

