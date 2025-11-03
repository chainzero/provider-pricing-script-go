package pricing

import (
	"fmt"
	"log"
	"os"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	dtypes "github.com/akash-network/akash-api/go/node/deployment/v1beta3"
	"github.com/akash-network/akash-api/go/node/types/v1beta3"
)

// Define default price targets as constants
const (
	DefaultCPUTarget         = 1.60
	DefaultMemoryTarget      = 0.80
	DefaultHDEphemeralTarget = 0.02
	DefaultHDPersHDDTarget   = 0.01
	DefaultHDPersSSDTarget   = 0.03
	DefaultHDPersNVMETarget  = 0.04
	DefaultEndpointTarget    = 0.05
	DefaultIPTarget          = 5.00

	AverageBlockTimeSeconds = 6.117 // Adjust as per the actual average block time
	DaysPerMonth            = 30.437
	BlocksPerMonth          = (60 / AverageBlockTimeSeconds) * 24 * 60 * DaysPerMonth
)

// CalculateRequestedResources computes the total requested resources from the GroupSpec
func CalculateRequestedResources(gSpec *dtypes.GroupSpec) ResourceRequests {
	var result ResourceRequests

	for _, resourceUnit := range gSpec.Resources {

		if resourceUnit.Resources.CPU != nil {
			cpuUnits := resourceUnit.Resources.CPU.Units.Val.Int64() // Get the CPU units in milliCPUs
			cpuCores := float64(cpuUnits) / 1000.0                   // Convert milliCPUs to CPU cores
			result.CPURequested += cpuCores * float64(resourceUnit.Count)
		}

		if resourceUnit.Resources.Memory != nil {
			memoryBytes := resourceUnit.Resources.Memory.Quantity.Val.Int64()
			memoryGB := float64(memoryBytes) / (1024.0 * 1024.0 * 1024.0) // Convert bytes to gigabytes
			result.MemoryRequested += memoryGB * float64(resourceUnit.Count)
		}

		for _, storage := range resourceUnit.Resources.Storage {
			// Default to using the 'name' field as the class if 'class' attribute is not found.
			storageClass := storage.Name

			// Look for 'class' in attributes to override the default if present.
			for _, attr := range storage.Attributes {
				if attr.Key == "class" {
					storageClass = attr.Value
					break
				}
			}

			storageBytes := storage.Quantity.Val.Int64()
			storageGB := storageBytes / (1024 * 1024 * 1024) // Convert bytes to gigabytes

			switch storageClass {
			case "ephemeral", "default":
				result.EphemeralStorageRequested += storageGB * int64(resourceUnit.Count)
			case "beta1":
				result.HDDPersStorageRequested += storageGB * int64(resourceUnit.Count)
			case "beta2":
				result.SSDPersStorageRequested += storageGB * int64(resourceUnit.Count)
			case "beta3":
				result.NVMePersStorageRequested += storageGB * int64(resourceUnit.Count)
			}
		}

		for _, endpoint := range resourceUnit.Resources.Endpoints {
			result.EndpointsRequested += int64(resourceUnit.Count) // Assuming 1 endpoint per resource unit count
			if endpoint.Kind == v1beta3.Endpoint_LEASED_IP {
				result.IPsRequested += int64(resourceUnit.Count) // Assuming 1 IP per resource unit count
			}
		}
	}

	return result
}

// GetEnvFloat gets an environment variable as a float, returning a default value if not set or invalid
func GetEnvFloat(envVar string, defaultValue float64) float64 {
	if val, ok := os.LookupEnv(envVar); ok {
		if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

// SetPriceTargets sets the price targets from environment variables or uses defaults
func SetPriceTargets() PriceTargets {
	gpuMappingsStr := os.Getenv("PRICE_TARGET_GPU_MAPPINGS") // Assuming this environment variable contains the mappings
	gpuMappings, err := ParseGPUPriceMappings(gpuMappingsStr)
	if err != nil {
		log.Fatalf("Error parsing GPU mappings: %v", err)
	}

	return PriceTargets{
		CPUTarget:         GetEnvFloat("PRICE_TARGET_CPU", DefaultCPUTarget),
		MemoryTarget:      GetEnvFloat("PRICE_TARGET_MEMORY", DefaultMemoryTarget),
		HDEphemeralTarget: GetEnvFloat("PRICE_TARGET_HD_EPHEMERAL", DefaultHDEphemeralTarget),
		HDPersHDDTarget:   GetEnvFloat("PRICE_TARGET_HD_PERS_HDD", DefaultHDPersHDDTarget),
		HDPersSSDTarget:   GetEnvFloat("PRICE_TARGET_HD_PERS_SSD", DefaultHDPersSSDTarget),
		HDPersNVMETarget:  GetEnvFloat("PRICE_TARGET_HD_PERS_NVME", DefaultHDPersNVMETarget),
		EndpointTarget:    GetEnvFloat("PRICE_TARGET_ENDPOINT", DefaultEndpointTarget),
		IPTarget:          GetEnvFloat("PRICE_TARGET_IP", DefaultIPTarget),
		GPUMappings:       gpuMappings,
	}
}

// CalculateTotalCostUsdTarget calculates the total cost in USD based on resource requests and price targets
func CalculateTotalCostUsdTarget(resourceRequests ResourceRequests, priceTargets PriceTargets) float64 {
	var totalCostUsdTarget float64

	cpuCost := float64(resourceRequests.CPURequested) * priceTargets.CPUTarget
	totalCostUsdTarget += cpuCost

	memoryCost := float64(resourceRequests.MemoryRequested) * priceTargets.MemoryTarget
	totalCostUsdTarget += memoryCost

	ephemeralStorageCost := float64(resourceRequests.EphemeralStorageRequested) * priceTargets.HDEphemeralTarget
	totalCostUsdTarget += ephemeralStorageCost

	hddPersStorageCost := float64(resourceRequests.HDDPersStorageRequested) * priceTargets.HDPersHDDTarget
	totalCostUsdTarget += hddPersStorageCost

	ssdPersStorageCost := float64(resourceRequests.SSDPersStorageRequested) * priceTargets.HDPersSSDTarget
	totalCostUsdTarget += ssdPersStorageCost

	nvmePersStorageCost := float64(resourceRequests.NVMePersStorageRequested) * priceTargets.HDPersNVMETarget
	totalCostUsdTarget += nvmePersStorageCost

	endpointCost := float64(resourceRequests.EndpointsRequested) * priceTargets.EndpointTarget
	totalCostUsdTarget += endpointCost

	ipCost := float64(resourceRequests.IPsRequested) * priceTargets.IPTarget
	totalCostUsdTarget += ipCost

	return totalCostUsdTarget
}

// CalculateBlockRates converts monthly USD costs to per-block rates
func CalculateBlockRates(totalCostUsdTarget float64, usdPerAkt float64, precision int) (float64, float64, string) {
	totalCostAktTarget := totalCostUsdTarget / usdPerAkt
	totalCostUaktTarget := totalCostAktTarget * 1000000 // Convert AKT to microAKT (uakt)

	ratePerBlockUakt := totalCostUaktTarget / BlocksPerMonth
	ratePerBlockUsd := totalCostUsdTarget / BlocksPerMonth

	// Format to the desired precision with 16 decimal places and append "uakt"
	totalCostUaktStr := fmt.Sprintf("%.*f", 16, ratePerBlockUakt) + "uakt"

	return ratePerBlockUakt, ratePerBlockUsd, totalCostUaktStr
}

// HandleDenomLogic processes the logic based on the received denom
func HandleDenomLogic(denom string, ratePerBlockUakt float64, ratePerBlockUsd float64, precision int, amount sdk.Dec) (string, error) {
	switch denom {
	case "uakt":
		if ratePerBlockUakt > amount.MustFloat64() { // Convert sdk.Dec to float64 for comparison
			return "", fmt.Errorf("requested rate is too low. min expected %.*f%s", precision, ratePerBlockUakt, denom)
		}
		return fmt.Sprintf("%.*f", precision, ratePerBlockUakt), nil

	case "ibc/12C6A0C374171B595A0A9E18B83FA09D295FB1F2D8C6DAA3AC28683471752D84",
		"ibc/170C677610AC31DF0904FFE09CD3B5C657492170E7E52372E48756B71E56F2F1":
		ratePerBlockUsdNormalized := ratePerBlockUsd * 1000000
		if ratePerBlockUsdNormalized > amount.MustFloat64() {
			return "", fmt.Errorf("requested rate is too low. min expected %.*f%s", precision, ratePerBlockUsdNormalized, denom)
		}
		return fmt.Sprintf("%.*f", precision, ratePerBlockUsdNormalized), nil

	default:
		return "", fmt.Errorf("denom is not supported: %s", denom)
	}
}

// RequestToBidPrice is the entry point to execute the bidding logic.
func RequestToBidPrice(request Request) error {
	fmt.Println("####Request: ", request)
	owner := request.Owner
	if owner == "" {
		return fmt.Errorf("request owner is not specified")
	}

	var denom string
	var amount sdk.Dec
	if request.GSpec != nil && len(request.GSpec.Resources) > 0 {
		denom = request.GSpec.Resources[0].Price.Denom
		amount = request.GSpec.Resources[0].Price.Amount
	}

	if SpecialPricing(owner) {
		log.Println("Special pricing activated")
		specialRate := "1.00"
		fmt.Printf("Special pricing rate per block (uakt): %s\n", specialRate)
		return nil
	}

	if err := CheckWhitelist(owner); err != nil {
		log.Printf("Whitelist check failed: %v", err)
		return fmt.Errorf("whitelist check failed: %v", err)
	}

	usdPerAkt, err := GetAKTPrice()
	if err != nil {
		log.Printf("Error getting AKT price: %v", err)
		return fmt.Errorf("error getting AKT price: %v", err)
	}

	if denom == "" || amount.IsZero() {
		fmt.Println("Price information is missing or incomplete")
		return fmt.Errorf("price information is missing or incomplete")
	}

	precision := request.PricePrecision
	if precision == 0 {
		precision = 6
	}

	if request.GSpec == nil {
		return fmt.Errorf("GroupSpec is nil in the request")
	}

	priceTargets := SetPriceTargets()
	maxGPUPrice := MaxGPUPrice(priceTargets.GPUMappings)
	totalGPUPrice := CalculateTotalGPUPrice(request.GSpec, priceTargets.GPUMappings, maxGPUPrice)
	resourceRequests := CalculateRequestedResources(request.GSpec)
	totalCostUsdTarget := CalculateTotalCostUsdTarget(resourceRequests, priceTargets) + totalGPUPrice

	// In RequestToBidPrice function
	_, _, finalRateStr := CalculateBlockRates(totalCostUsdTarget, usdPerAkt, precision)

	if err != nil {
		log.Println(err)
		return err
	}

	// Now, finalRateStr already has the "uakt" suffix and the correct number of decimal places
	fmt.Printf("Total cost per block (uakt, formatted): %s\n", finalRateStr)

	fmt.Printf("Total cost in USD: %.2f/month\n", totalCostUsdTarget)

	return nil
}
