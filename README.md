# Akash Pricing Script

A Go implementation of the Akash Network pricing script for calculating deployment bid prices. This is a direct port of the [price_script_generic.sh](https://github.com/akash-network/helm-charts/blob/main/charts/akash-provider/scripts/price_script_generic.sh) bash script, offering improved performance, type safety, and easier maintenance.

Providers can use this to determine appropriate bid prices based on resource requirements including CPU, memory, storage, GPUs, IPs, and endpoints.

## Project Structure

```
akash-pricing-script/
├── README.md                    # This file
├── go.mod                       # Go module definition
├── pricing.go                   # Core pricing calculations
├── types.go                     # Data structures
├── gpu.go                       # GPU pricing logic
├── cache.go                     # AKT price caching
├── whitelist.go                 # Whitelist and special pricing
├── cmd/
│   └── pricing-tool/           # Standalone CLI tool / Binary
│       └── main.go
└── examples/
    └── sample-deployment.json   # Example deployment input
```

## Three Deployment Strategies

This project supports three different ways to deploy pricing logic on Akash Providers:

### 1. 🚀 As a Compiled Binary (Recommended - Drop-in Replacement)

**The Akash Provider accepts ANY executable** (bash script OR compiled binary) as a bid price script. This compiled Go binary is a drop-in replacement for the bash script with better performance.

**Build for Linux** (most providers run on Linux):

```bash
# Clone and build
git clone https://github.com/akash-network/pricing-script.git
cd pricing-script
GOOS=linux GOARCH=amd64 go build -o pricing-tool cmd/pricing-tool/main.go

# Verify the binary works
cat examples/sample-deployment.json | ./pricing-tool
```

**Deploy via Helm** (same process as bash script):

```bash
# Download or build the binary first
# Then deploy it the exact same way as the bash script:
helm upgrade akash-provider akash/provider -n akash-services -f provider.yaml \
  --set bidpricescript="$(cat pricing-tool | openssl base64 -A)"
```

**Benefits**:
- ✅ Drop-in replacement for `price_script_generic.sh`
- ✅ Better performance (no shell overhead, faster execution)
- ✅ Type safety and easier debugging
- ✅ Same deployment method as bash script
- ✅ No Provider code changes required

**Use case**: Production providers wanting better performance without modifying Provider code

### 2. 📚 As a Go Library (Deep Integration)

Import this package directly into the Akash Provider codebase for the deepest integration:

```go
import (
    "github.com/akash-network/pricing-script"
)

func calculateBid(request bidRequest) error {
    pricingRequest := pricing.Request{
        Owner: request.Owner,
        GSpec: request.GroupSpec,
        PricePrecision: 6,
    }
    
    return pricing.RequestToBidPrice(pricingRequest)
}
```

**Benefits**:
- ✅ No external script/binary needed
- ✅ Direct function calls (lowest latency)
- ✅ Easier to extend and customize

**Drawbacks**:
- ❌ Requires modifying Provider source code
- ❌ Needs Provider recompilation for updates

**Use case**: Future Provider versions with native Go pricing integration

### 3. 🔧 As a CLI Tool (Testing & Development)

Run locally for testing pricing calculations:

```bash
# Build the tool
cd cmd/pricing-tool
go build -o pricing-tool

# Test with sample data
cat ../../examples/sample-deployment.json | ./pricing-tool

# Test with custom pricing
export PRICE_TARGET_CPU=2.00
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=120.00,a100=200.00"
cat ../../examples/sample-deployment.json | ./pricing-tool
```

**Use case**: Testing pricing, debugging configurations, manual calculations

## Quick Start

### For Providers (Binary Deployment)

```bash
# 1. Clone and build for Linux
git clone https://github.com/akash-network/pricing-script.git
cd pricing-script
GOOS=linux GOARCH=amd64 go build -o pricing-tool cmd/pricing-tool/main.go

# 2. Configure your pricing (optional - uses defaults if not set)
export PRICE_TARGET_CPU=1.60
export PRICE_TARGET_MEMORY=0.80
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=120.00,a100=200.00"

# 3. Test the binary locally (optional)
cat examples/sample-deployment.json | ./pricing-tool

# 4. Deploy to your provider via Helm
helm upgrade akash-provider akash/provider -n akash-services -f provider.yaml \
  --set bidpricescript="$(cat pricing-tool | openssl base64 -A)"
```

### For Library Integration

```bash
go get github.com/akash-network/pricing-script
```

### For Local Testing

```bash
git clone https://github.com/akash-network/pricing-script.git
cd pricing-script/cmd/pricing-tool
go build -o pricing-tool
cat ../../examples/sample-deployment.json | ./pricing-tool
```

## Configuration

Configure pricing through environment variables:

### Resource Pricing (USD per unit per month)

```bash
export PRICE_TARGET_CPU=1.60              # Per CPU core
export PRICE_TARGET_MEMORY=0.80           # Per GB RAM
export PRICE_TARGET_HD_EPHEMERAL=0.02     # Per GB ephemeral storage
export PRICE_TARGET_HD_PERS_HDD=0.01      # Per GB HDD persistent storage
export PRICE_TARGET_HD_PERS_SSD=0.03      # Per GB SSD persistent storage
export PRICE_TARGET_HD_PERS_NVME=0.04     # Per GB NVMe persistent storage
export PRICE_TARGET_ENDPOINT=0.05         # Per endpoint
export PRICE_TARGET_IP=5.00               # Per IP address
```

### GPU Pricing

GPU pricing uses a mapping format: `model=price,model.vram=price`

```bash
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=120.00,rtx4090.24gi=150.00,a100=200.00"
```

The script will match GPUs in this order:
1. `model.vram.interface` (most specific)
2. `model.vram`
3. `model` (least specific)
4. Falls back to max price (100.00) if no match

### Optional Configuration

```bash
# Whitelist URL (leave empty to disable whitelist checking)
export WHITELIST_URL="https://example.com/whitelist.txt"

# Owner address (for provider integration)
export AKASH_OWNER="akash1..."
```

## CLI Tool Usage

### Basic Example

```bash
echo '{
  "resources": [{
    "memory": 2147483648,
    "cpu": 1000,
    "storage": [{"class": "ephemeral", "size": 10737418240}],
    "count": 1,
    "endpoint_quantity": 1,
    "ip_lease_quantity": 0
  }],
  "price": {"denom": "uakt", "amount": "100.00"}
}' | ./pricing-tool
```

### With GPU Pricing

```bash
export PRICE_TARGET_GPU_MAPPINGS="rtx4090.24gi=150.00"
cat examples/sample-deployment.json | ./pricing-tool
```

### Output Example

```
CPU Requested: 1.00 cores
Memory Requested: 2.000000000000 GB
Ephemeral Storage Requested: 10.000000000000 GB
IPs Requested: 0
Endpoints Requested: 1
CPU Price: $1.60 per core/month
Memory Price: $0.80 per GB/month
Total GPU Price: $150.00
Total Monthly Cost: $153.45
```

## Features

### Resource Calculations
- **CPU**: Measured in cores (1000 millicores = 1 core)
- **Memory**: Measured in GB
- **Storage**: Supports ephemeral, HDD (beta1), SSD (beta2), NVMe (beta3)
- **GPU**: Flexible model/VRAM/interface matching
- **Networking**: IP leases and endpoints

### AKT Price Integration
- Fetches current AKT/USD price from APIs
- Caches price for 60 minutes
- Supports primary (Osmosis) and fallback (CoinGecko) APIs
- Converts monthly USD costs to per-block uAKT rates

### Whitelist Support
- Optional whitelist checking via URL
- Caches whitelist for 10 minutes
- Special pricing for designated accounts

### Block Rate Calculations
- Uses actual Akash block time (6.117 seconds)
- Converts monthly costs to per-block rates
- Supports multiple denoms (uakt, IBC tokens)

## Building for Different Platforms

### Linux (Production - Most Providers)

```bash
# Standard build
GOOS=linux GOARCH=amd64 go build -o pricing-tool cmd/pricing-tool/main.go

# Optimized build (smaller binary)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o pricing-tool cmd/pricing-tool/main.go
```

### macOS (Local Testing)

```bash
# Intel
GOOS=darwin GOARCH=amd64 go build -o pricing-tool-macos cmd/pricing-tool/main.go

# Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o pricing-tool-macos cmd/pricing-tool/main.go
```

### Windows

```bash
GOOS=windows GOARCH=amd64 go build -o pricing-tool.exe cmd/pricing-tool/main.go
```

## Relationship to Bash Script

This Go implementation is a **direct port** of the canonical bash script:
- [price_script_generic.sh](https://github.com/akash-network/helm-charts/blob/main/charts/akash-provider/scripts/price_script_generic.sh)

**Key features ported from bash to Go:**
- ✅ Same pricing calculations and formulas
- ✅ Same AKT price fetching (Osmosis/CoinGecko APIs)
- ✅ Same block time (6.117s) and blocks per month (429,909)
- ✅ Same GPU model matching logic with fallbacks
- ✅ Same whitelist checking with caching
- ✅ Same special account pricing
- ✅ Same denom support (uakt, IBC USDC)

**Advantages of Go version:**
- 🚀 Faster execution (compiled vs interpreted)
- 🔒 Type safety and compile-time checks
- 🧪 Easier to test and maintain
- 📦 Single binary (no dependency on curl, jq, bc, mawk)

Providers can use either the bash script or this Go binary - they're functionally equivalent!

## Development

### Running Tests

```bash
go test ./...
```

### Code Structure

- `pricing.go` - Core pricing logic and orchestration
- `gpu.go` - GPU model parsing and price matching
- `cache.go` - AKT price fetching and caching
- `whitelist.go` - Whitelist checking and special pricing
- `types.go` - Data structures
- `cmd/pricing-tool/main.go` - Standalone binary (reads JSON from stdin)

## JSON Input Format

The CLI tool expects JSON with this structure:

```json
{
  "resources": [
    {
      "memory": 2147483648,
      "cpu": 1000,
      "gpu": {
        "units": 1,
        "attributes": {
          "vendor": {
            "model": "rtx4090",
            "ram": "24gi",
            "interface": "pcie"
          }
        }
      },
      "storage": [
        {
          "class": "ephemeral",
          "size": 10737418240
        }
      ],
      "count": 1,
      "endpoint_quantity": 1,
      "ip_lease_quantity": 0
    }
  ],
  "price": {
    "denom": "uakt",
    "amount": "100.00"
  },
  "price_precision": 6
}
```

## FAQ

### Why use the Go binary instead of the bash script?

**Performance**: The compiled Go binary executes faster than the bash script (no shell interpretation overhead).

**Reliability**: Single static binary with no external dependencies (no need for curl, jq, bc, mawk to be installed).

**Maintainability**: Type-safe code that's easier to test, debug, and extend.

**Compatibility**: Drop-in replacement - deploys exactly the same way as the bash script.

### Can I still use the bash script?

Yes! The bash script and Go binary are functionally equivalent. Use whichever you prefer. The Akash Provider accepts any executable as a bid price script.

### How does the Provider execute the binary/script?

The Provider base64-encodes your script/binary via Helm, then executes it for each bid request:
1. Provider passes deployment specs as JSON to stdin
2. Script/binary calculates the bid price
3. Script/binary outputs the price to stdout
4. Provider uses that price for the bid

Both the bash script and Go binary follow this same contract.

### What environment variables does it use?

The Go binary uses the **exact same environment variables** as the bash script:
- `PRICE_TARGET_CPU`, `PRICE_TARGET_MEMORY`, etc. - Pricing configuration
- `PRICE_TARGET_GPU_MAPPINGS` - GPU model pricing
- `WHITELIST_URL` - Optional whitelist URL
- `AKASH_OWNER` - Tenant address (passed by Provider)
- `DEBUG_BID_SCRIPT` - Enable debug logging

### How do I migrate from bash to Go?

Simply rebuild with the new binary:

```bash
# 1. Build the binary
GOOS=linux GOARCH=amd64 go build -o pricing-tool cmd/pricing-tool/main.go

# 2. Deploy (same as before, just change the filename)
helm upgrade akash-provider akash/provider -n akash-services -f provider.yaml \
  --set bidpricescript="$(cat pricing-tool | openssl base64 -A)"
```

Your existing `provider.yaml` configuration and environment variables remain unchanged!

## Contributing

Contributions are welcome! Please ensure:
1. Code follows Go conventions
2. Tests pass
3. Documentation is updated
4. Commit messages are clear
5. Changes maintain parity with the bash script behavior

## License

Apache 2.0

## Support

For issues or questions:
- GitHub Issues: https://github.com/akash-network/pricing-script/issues
- Akash Discord: https://discord.akash.network

