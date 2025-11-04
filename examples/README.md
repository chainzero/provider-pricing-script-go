# Example Deployment Files

This directory contains sample deployment JSON files for testing the pricing-tool.

## Files

### Basic Examples

**`sample-deployment.json`** - GPU deployment with RTX 4090
- Use case: Testing GPU pricing with specific model and VRAM
- Resources: 1 CPU, 2GB RAM, 10GB storage, 1x RTX 4090 24GB

**`cpu-only-deployment.json`** - Simple CPU deployment without GPU
- Use case: Testing basic resource pricing (no GPU)
- Resources: 2 CPU, 4GB RAM, 20GB storage

### Advanced Examples

**`multi-storage-deployment.json`** - Multiple storage classes
- Use case: Testing different storage types and pricing
- Resources: 4 CPU, 8GB RAM, ephemeral + SSD + NVMe storage, 2 endpoints, 1 IP

**`multi-gpu-deployment.json`** - Multiple GPUs with full specs
- Use case: Testing GPU count, VRAM, and interface detection
- Resources: 8 CPU, 16GB RAM, 2x A100 80GB SXM4

**`old-format-deployment.json`** - Legacy format without price field
- Use case: Testing backward compatibility
- Resources: 1 CPU, 2GB RAM, 10GB storage (no GPU)

## Usage

```bash
# Build the tool first
go build -o pricing-tool cmd/pricing-tool/main.go

# Test any example
cat examples/sample-deployment.json | ./pricing-tool

# With custom pricing
export PRICE_TARGET_GPU_MAPPINGS="rtx4090=150.00"
cat examples/sample-deployment.json | ./pricing-tool
```

## Creating Custom Test Files

### Template

```json
{
  "resources": [
    {
      "memory": 2147483648,          // bytes (2GB)
      "cpu": 1000,                   // millicores (1 CPU)
      "gpu": {                       // optional
        "units": 1,
        "attributes": {
          "vendor": {
            "nvidia": {              // or "amd"
              "model": "rtx4090",
              "ram": "24gi",         // optional
              "interface": "pcie"    // optional
            }
          }
        }
      },
      "storage": [
        {
          "class": "ephemeral",      // or beta1, beta2, beta3
          "size": 10737418240        // bytes (10GB)
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

### Storage Classes

- `ephemeral` or `default` - Temporary storage (deleted on teardown)
- `beta1` - HDD persistent storage
- `beta2` - SSD persistent storage  
- `beta3` - NVMe persistent storage

### Units

- **Memory**: bytes (e.g., 2147483648 = 2GB)
- **CPU**: millicores (e.g., 1000 = 1 CPU core)
- **Storage**: bytes (e.g., 10737418240 = 10GB)

### Conversions

```bash
# Memory
1 GB = 1,073,741,824 bytes
2 GB = 2,147,483,648 bytes
4 GB = 4,294,967,296 bytes
8 GB = 8,589,934,592 bytes

# Storage
10 GB = 10,737,418,240 bytes
20 GB = 21,474,836,480 bytes
50 GB = 53,687,091,200 bytes
100 GB = 107,374,182,400 bytes

# CPU
0.5 CPU = 500 millicores
1 CPU = 1,000 millicores
2 CPU = 2,000 millicores
4 CPU = 4,000 millicores
```

## See Also

- [TESTING.md](../TESTING.md) - Comprehensive testing guide
- [README.md](../README.md) - Project documentation

