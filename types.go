package pricing

import (
	"encoding/json"

	dtypes "pkg.akt.dev/go/node/deployment/v1beta4"
)

// ResourceRequests holds the calculated resource requirements
type ResourceRequests struct {
	CPURequested              float64
	MemoryRequested           float64
	EphemeralStorageRequested int64
	HDDPersStorageRequested   int64
	SSDPersStorageRequested   int64
	NVMePersStorageRequested  int64
	IPsRequested              int64
	EndpointsRequested        int64
}

// PriceTargets holds the pricing configuration
type PriceTargets struct {
	CPUTarget         float64
	MemoryTarget      float64
	HDEphemeralTarget float64
	HDPersHDDTarget   float64
	HDPersSSDTarget   float64
	HDPersNVMETarget  float64
	EndpointTarget    float64
	IPTarget          float64
	GPUMappings       map[string]float64
}

// Request represents a bid request from the Akash network
type Request struct {
	Owner          string
	GSpec          *dtypes.GroupSpec
	PricePrecision int
}

// DeploymentOrder represents the structure of the data received from the Akash Provider.
type DeploymentOrder struct {
	Price          *Price          `json:"price"`
	PricePrecision int             `json:"price_precision"`
	Resources      json.RawMessage `json:"resources"`
}

// Price represents the price structure in the deployment order.
type Price struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}
