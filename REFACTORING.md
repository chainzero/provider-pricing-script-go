# Refactoring Guide: Migrating to chain-sdk

This document outlines the refactoring effort to modernize the codebase by using official Akash API types from the new `chain-sdk` repository.

## Executive Summary

**Goal**: Replace custom JSON structs with official Akash protobuf-generated types from `chain-sdk`

**Benefits**:
- ✅ Type safety guaranteed by official Akash types
- ✅ Automatic API compatibility
- ✅ Future-proof against API changes
- ✅ Easier maintenance
- ✅ Better documentation (types match official Akash docs)

## Phase 1: Migrate Library to chain-sdk (COMPLETED ✅)

### What Changed

**Old dependency:**
```go
github.com/akash-network/akash-api/go v0.0.0-20240610212142-975a9f1c0ba7
```

**New dependency:**
```go
github.com/akash-network/chain-sdk/go v0.1.0
```

### Files Updated

1. **`go.mod`** - Updated dependency
2. **`pricing.go`** - Updated imports
3. **`gpu.go`** - Updated imports
4. **`types.go`** - Updated imports

### Import Changes

**Before:**
```go
import (
    dtypes "github.com/akash-network/akash-api/go/node/deployment/v1beta3"
    "github.com/akash-network/akash-api/go/node/types/v1beta3"
)
```

**After:**
```go
import (
    dtypes "github.com/akash-network/chain-sdk/go/node/deployment/v1beta3"
    "github.com/akash-network/chain-sdk/go/node/types/v1beta3"
)
```

### API Compatibility

The `chain-sdk` repository contains the **same protobuf-generated Go code**, just in a new location. The types are identical, so no code logic changes were required - only import paths.

**Key types used:**
- `dtypes.GroupSpec` - Deployment group specification
- `dtypes.ResourceUnit` - Resource requirements per service
- `v1beta3.Endpoint` - Network endpoint configuration
- `sdk.Dec` - Cosmos SDK decimal type for prices

### Testing After Migration

```bash
# Update dependencies
go mod tidy

# Ensure everything compiles
go build ./...

# Run existing tests
go test ./...

# Test the library
cd cmd/pricing-tool
go build -o pricing-tool main.go
```

Verify the binary still works:
```bash
cat ../../examples/sample-deployment.json | ./pricing-tool
```

---

## Phase 2: Refactor CLI Tool (NEXT)

### Current State

The CLI tool (`cmd/pricing-tool/main.go`) uses **custom structs** to parse JSON:

```go
type GPUVendor struct {
	Model     string `json:"model"`
	RAM       string `json:"ram"`
	Interface string `json:"interface"`
}

type Resource struct {
	Memory           int64     `json:"memory"`
	CPU              int       `json:"cpu"`
	GPU              *GPU      `json:"gpu,omitempty"`
	Storage          []Storage `json:"storage"`
	// ...
}

type DeploymentData struct {
	Resources      []Resource `json:"resources"`
	Price          *Price     `json:"price,omitempty"`
	PricePrecision int        `json:"price_precision"`
}
```

### Target State

Use official Akash types from `chain-sdk`:

```go
import (
	dtypes "github.com/akash-network/chain-sdk/go/node/deployment/v1beta3"
)

// Parse directly into Akash types
var gspec dtypes.GroupSpec
json.Unmarshal(inputData, &gspec)
```

### Why This Refactor?

**Problem with custom structs:**
- ❌ Duplicate type definitions (custom vs. official)
- ❌ Risk of struct mismatch with actual Provider data
- ❌ Manual maintenance when API evolves
- ❌ No compile-time guarantee of compatibility

**Benefits of official types:**
- ✅ Single source of truth (chain-sdk)
- ✅ Guaranteed API compatibility
- ✅ Auto-updates with dependency updates
- ✅ Better IDE autocomplete and documentation
- ✅ Matches what the Provider actually sends

### JSON Format Compatibility

The Provider sends JSON that matches the protobuf structure. The JSON field names use snake_case (protobuf convention) vs. our custom structs using camelCase.

**Provider JSON example:**
```json
{
  "resources": [
    {
      "cpu": {"units": {"val": "100"}},
      "memory": {"quantity": {"val": "268435456"}},
      "storage": [{"name": "default", "quantity": {"val": "268435456"}}],
      "gpu": {"units": {"val": "1"}},
      "count": 1
    }
  ],
  "price": {"denom": "uakt", "amount": "100000"}
}
```

**Key differences:**
- Nested structure: `cpu.units.val` vs flat `cpu`
- Field names: `quantity` vs `size`
- Type wrappers: protobuf uses wrapper types

### Refactoring Strategy

#### Option A: Parse Provider JSON Directly (Ideal)

The Provider passes JSON that should map to Akash protobufs. We should be able to unmarshal directly:

```go
// Simplified main.go
func main() {
	inputData, _ := io.ReadAll(os.Stdin)
	
	// Parse as DeploymentOrder (matches Provider format)
	var order struct {
		Resources []dtypes.ResourceUnit `json:"resources"`
		Price     *sdk.DecCoin          `json:"price"`
		Precision int                   `json:"price_precision"`
	}
	
	json.Unmarshal(inputData, &order)
	
	// Convert to library format and calculate
	// ...
}
```

#### Option B: Adapter Pattern (Pragmatic)

If direct parsing doesn't work due to JSON format differences, create adapters:

```go
// Parse custom format, convert to Akash types
var customData DeploymentData
json.Unmarshal(inputData, &customData)

// Convert to official types
gspec := convertToGroupSpec(customData)

// Use library with official types
price := pricing.CalculatePrice(gspec)
```

#### Option C: Keep Custom Structs, Validate Against Official (Compromise)

Keep custom structs for JSON parsing, but add validation:

```go
// Ensure our custom types match official structure
func validateStructure(custom DeploymentData) error {
	// Convert and verify fields match
	// This at least documents the relationship
}
```

### Implementation Plan

**Step 1: Analyze Provider JSON Format**

Capture actual JSON from Provider and document the structure:

```bash
# From provider logs
kubectl logs akash-provider-0 -n akash-services | grep "reservation requested"
```

**Step 2: Attempt Direct Parsing**

Try unmarshaling Provider JSON directly into `dtypes.ResourceUnit`:

```go
var resources []dtypes.ResourceUnit
err := json.Unmarshal(resourcesJSON, &resources)
```

**Step 3: Identify Gaps**

Document any fields that don't map directly, and why.

**Step 4: Choose Strategy**

Based on parsing results:
- If direct parsing works → Use Option A
- If format differences exist → Use Option B
- If too complex → Use Option C temporarily

**Step 5: Implement and Test**

- Refactor `cmd/pricing-tool/main.go`
- Test with all example JSON files
- Verify output matches current behavior
- Test on provider with real deployments

### Testing Strategy

**Unit Tests:**
```go
func TestDirectParsing(t *testing.T) {
	// Test that Provider JSON parses into Akash types
	jsonData := `{...}`
	var resources []dtypes.ResourceUnit
	err := json.Unmarshal([]byte(jsonData), &resources)
	assert.NoError(t, err)
	assert.Equal(t, expectedCPU, resources[0].Resources.CPU.Units)
}
```

**Integration Tests:**
```bash
# Test all example files produce same output
for file in examples/*.json; do
  echo "Testing $file"
  ./pricing-tool-old < $file > old-output.txt
  ./pricing-tool-new < $file > new-output.txt
  diff old-output.txt new-output.txt || echo "MISMATCH in $file"
done
```

**Provider Test:**
```bash
# Deploy refactored version to test provider
# Monitor bids to ensure prices match
kubectl logs -f akash-provider-0 | grep "submitting fulfillment"
```

---

## Phase 3: Enhanced Type Safety (FUTURE)

### Goals

1. **Remove all custom type definitions** from `cmd/pricing-tool/main.go`
2. **Use chain-sdk types throughout** the codebase
3. **Add validation** using protobuf validation rules
4. **Generate types from .proto files** if custom extensions needed

### Potential Enhancements

**Validation:**
```go
import "github.com/akash-network/chain-sdk/go/node/deployment/v1beta3"

func validateDeployment(gspec *dtypes.GroupSpec) error {
	return gspec.ValidateBasic() // Use built-in validation
}
```

**Type Conversions:**
```go
// Helper functions for common conversions
func resourceToPrice(r *dtypes.ResourceUnit) float64 {
	// Use official types with confidence
	cpuUnits := r.Resources.CPU.Units.Val.Uint64()
	// ...
}
```

**Proto Extensions:**
If we need custom types, extend via protobuf:

```protobuf
syntax = "proto3";
package pricing.v1;

import "akash/deployment/v1beta3/resourceunit.proto";

message PricingRequest {
  akash.deployment.v1beta3.GroupSpec group = 1;
  string owner = 2;
  int32 precision = 3;
}
```

---

## Migration Checklist

### Phase 1: Library (✅ COMPLETED)
- [x] Update `go.mod` to use `chain-sdk`
- [x] Update imports in `pricing.go`
- [x] Update imports in `gpu.go`
- [x] Update imports in `types.go`
- [x] Run `go mod tidy`
- [x] Verify compilation
- [x] Document changes

### Phase 2: CLI Tool (🔄 IN PROGRESS)
- [ ] Analyze Provider JSON format
- [ ] Document JSON structure differences
- [ ] Choose refactoring strategy (A, B, or C)
- [ ] Implement JSON parsing with Akash types
- [ ] Update `cmd/pricing-tool/main.go`
- [ ] Add unit tests for parsing
- [ ] Test with all example files
- [ ] Validate output matches current behavior
- [ ] Deploy to test provider
- [ ] Monitor production bids

### Phase 3: Enhancement (📋 PLANNED)
- [ ] Add protobuf validation
- [ ] Create helper functions
- [ ] Add comprehensive tests
- [ ] Document type usage patterns
- [ ] Consider proto extensions if needed

---

## Benefits Realized

### Immediate (Phase 1)
- ✅ Using official Akash dependency
- ✅ Future-proof against API location changes
- ✅ Better alignment with Akash ecosystem

### After Phase 2
- ✅ Reduced code duplication
- ✅ Guaranteed API compatibility
- ✅ Easier debugging (types match docs)
- ✅ Safer refactoring (compiler catches mismatches)

### After Phase 3
- ✅ Production-grade type safety
- ✅ Validation built-in
- ✅ Extensible architecture
- ✅ Best practices throughout

---

## Risk Management

### Risks

1. **JSON Format Mismatch**: Provider JSON might not map directly to protobuf types
   - **Mitigation**: Analyze actual Provider JSON first, use adapter if needed

2. **Breaking Changes**: `chain-sdk` might introduce breaking changes
   - **Mitigation**: Pin to specific version, test thoroughly before updating

3. **Performance**: Protobuf unmarshaling might be slower than custom structs
   - **Mitigation**: Benchmark, but pricing calculation is not latency-critical

4. **Complexity**: Protobuf types can be complex to work with
   - **Mitigation**: Create helper functions, good documentation

### Rollback Plan

If refactoring causes issues:

1. **Git revert** to previous version
2. **Keep custom structs** temporarily
3. **Gradual migration** (Option C) instead of full rewrite
4. **Document blockers** for future attempts

---

## Resources

- **chain-sdk repository**: https://github.com/akash-network/chain-sdk
- **Protobuf definitions**: https://github.com/akash-network/chain-sdk/tree/main/proto
- **Generated Go code**: https://github.com/akash-network/chain-sdk/tree/main/go
- **Akash API docs**: https://docs.akash.network/

---

## Next Steps

1. ✅ Complete Phase 1 (library migration to chain-sdk)
2. 🔄 Run `go mod tidy` to download dependencies
3. 🔄 Test compilation and existing functionality
4. 📋 Begin Phase 2: Analyze Provider JSON format
5. 📋 Choose refactoring strategy for CLI tool
6. 📋 Implement Phase 2 changes

**Current Status**: Phase 1 complete, ready to test and proceed to Phase 2.

---

## Questions for Discussion

1. **JSON Format**: Do we have documented examples of actual Provider JSON?
2. **Testing Access**: Can we capture real Provider JSON for testing?
3. **Version Pinning**: Should we pin to a specific chain-sdk version or use latest?
4. **Timeline**: Is there urgency, or can we take incremental approach?
5. **Custom Types**: Are there any custom extensions we need beyond standard Akash types?

