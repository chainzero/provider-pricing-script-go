package pricing

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	dtypes "github.com/akash-network/akash-api/go/node/deployment/v1beta3"
)

// ParseGPUPriceMappings parses a string of GPU model to price mappings and returns a map
func ParseGPUPriceMappings(mappingStr string) (map[string]float64, error) {
	gpuMappings := make(map[string]float64)

	// Return an empty map if the input string is empty, avoiding an error
	if mappingStr == "" {
		return gpuMappings, nil
	}

	pairs := strings.Split(mappingStr, ",")
	for _, pair := range pairs {
		// Continue with the next iteration if the pair is empty
		if pair == "" {
			continue
		}
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid GPU mapping: %s", pair)
		}

		key := kv[0]
		value, err := strconv.ParseFloat(kv[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid GPU price for %s: %v", key, err)
		}

		gpuMappings[key] = value
	}

	return gpuMappings, nil
}

// MaxGPUPrice returns the maximum GPU price from the mappings or a default value
func MaxGPUPrice(gpuMappings map[string]float64) float64 {
	maxPrice := 100.0 // Default value
	for _, price := range gpuMappings {
		if price > maxPrice {
			maxPrice = price
		}
	}
	return maxPrice
}

// CalculateTotalGPUPrice calculates the total GPU price based on the GroupSpec and GPU price mappings
func CalculateTotalGPUPrice(gSpec *dtypes.GroupSpec, gpuMappings map[string]float64, maxGPUPrice float64) float64 {
	totalGPUPrice := 0.0

	for _, resourceUnit := range gSpec.Resources {
		if resourceUnit.Resources.GPU != nil {
			count := float64(resourceUnit.Count)
			gpuUnits := float64(resourceUnit.Resources.GPU.Units.Val.Int64())

			var model, vram, interfaceType string
			// Parse GPU attributes to extract model, vram, and interface
			for _, attr := range resourceUnit.Resources.GPU.Attributes {
				parts := strings.Split(attr.Key, "/")
				for i, part := range parts {
					switch part {
					case "model":
						if i+1 < len(parts) {
							model = parts[i+1]
						}
					case "ram":
						if i+1 < len(parts) {
							vram = parts[i+1]
						}
					case "interface":
						if i+1 < len(parts) {
							interfaceType = parts[i+1]
						}
					}
				}
			}

			// Construct the key for price lookup
			gpuKey := model
			if vram != "" {
				gpuKey += "." + vram
			}
			if interfaceType != "" {
				gpuKey += "." + interfaceType
			}

			// Find the best price matching the complete key or fallbacks
			price, found := gpuMappings[gpuKey]
			if !found && interfaceType != "" {
				// Try model.vram or model
				gpuKey = model + "." + vram
				price, found = gpuMappings[gpuKey]
				if !found {
					// Try model only
					gpuKey = model
					price, found = gpuMappings[gpuKey]
					if !found {
						price = maxGPUPrice
					}
				}
			}

			totalGPUPrice += count * gpuUnits * price
			log.Printf("GPU Pricing: Model=%s, VRAM=%s, Interface=%s, Units=%f, Price=%f, Total=%f",
				model, vram, interfaceType, gpuUnits, price, count*gpuUnits*price)
		}
	}

	return totalGPUPrice
}
