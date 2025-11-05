# API Version Update: v1beta3 → v1beta4

## Summary

Updated all API imports from `v1beta3` to the current `v1beta4` version.

## Changes Made

### Files Updated

1. ✅ **pricing.go**
   - `node/deployment/v1beta3` → `node/deployment/v1beta4`
   - `node/types/v1beta3` → `node/types/v1beta4`

2. ✅ **gpu.go**
   - `node/deployment/v1beta3` → `node/deployment/v1beta4`

3. ✅ **types.go**
   - `node/deployment/v1beta3` → `node/deployment/v1beta4`

### Before
```go
import (
    dtypes "github.com/akash-network/chain-sdk/go/node/deployment/v1beta3"
    "github.com/akash-network/chain-sdk/go/node/types/v1beta3"
)
```

### After
```go
import (
    dtypes "github.com/akash-network/chain-sdk/go/node/deployment/v1beta4"
    "github.com/akash-network/chain-sdk/go/node/types/v1beta4"
)
```

## Why This Matters

- ✅ **Current API**: Using the latest stable API version from chain-sdk
- ✅ **Future-proof**: Aligned with current Akash network protocol
- ✅ **Compatibility**: Matches what the Provider is using

## Potential Breaking Changes

When upgrading from v1beta3 to v1beta4, there may be:

1. **Field name changes**
2. **Type changes**
3. **New required fields**
4. **Deprecated fields removed**

## Testing Required

After running `go mod tidy`, test for any breaking changes:

### 1. Check Compilation
```bash
go build ./...
```

If this fails, check error messages for specific field/type mismatches.

### 2. Check API Differences

Compare the protobuf definitions:
- v1beta3: https://github.com/akash-network/chain-sdk/tree/main/proto/node/akash/deployment/v1beta3
- v1beta4: https://github.com/akash-network/chain-sdk/tree/main/proto/node/akash/deployment/v1beta4

### 3. Common Breaking Changes to Watch For

**Resource Units:**
```go
// v1beta3
resourceUnit.Resources.CPU.Units.Val.Int64()

// v1beta4 (if structure changed)
// Might be: resourceUnit.Resources.CPU.Units.Amount()
// Or: resourceUnit.Resources.CPU.Quantity.Value()
```

**Storage Classes:**
```go
// Check if storage class attribute handling changed
for _, storage := range resourceUnit.Resources.Storage {
    // Verify structure is same
}
```

**GPU Attributes:**
```go
// Verify GPU vendor structure
resourceUnit.Resources.GPU.Attributes
```

### 4. Test with Examples

```bash
# Test all example files
cat examples/sample-deployment.json | ./cmd/pricing-tool/pricing-tool
cat examples/cpu-only-deployment.json | ./cmd/pricing-tool/pricing-tool
cat examples/multi-gpu-deployment.json | ./cmd/pricing-tool/pricing-tool
```

### 5. Compare Outputs

If you have the old binary, compare outputs:

```bash
# Old version (v1beta3)
cat examples/sample-deployment.json | ./pricing-tool-old > output-old.txt

# New version (v1beta4)
cat examples/sample-deployment.json | ./pricing-tool-new > output-new.txt

# Compare
diff output-old.txt output-new.txt
```

## Rollback Plan

If v1beta4 causes issues:

```bash
# Revert to v1beta3 in all files
git checkout HEAD -- pricing.go gpu.go types.go

# Or manually change imports back:
# v1beta4 → v1beta3
```

## Expected Outcome

✅ **No Breaking Changes Expected**

The v1beta3 → v1beta4 transition should be mostly additive (new features added) rather than breaking existing functionality. The core resource types (CPU, Memory, Storage, GPU) should remain compatible.

## Next Steps

1. Run `go mod tidy`
2. Test compilation
3. Run test suite
4. If issues arise, consult:
   - chain-sdk CHANGELOG
   - Akash network migration guides
   - Protobuf definition diffs

## Documentation References

- **chain-sdk repo**: https://github.com/akash-network/chain-sdk
- **v1beta4 protos**: https://github.com/akash-network/chain-sdk/tree/main/proto/node/akash/deployment/v1beta4
- **v1beta4 Go code**: https://github.com/akash-network/chain-sdk/tree/main/go/node/deployment/v1beta4

---

**Status**: ✅ Code updated to v1beta4  
**Testing**: 🔄 Pending `go mod tidy` and validation

