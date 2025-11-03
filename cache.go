package pricing

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// GetAKTPrice fetches the current price of AKT from the APIs, caching it.
func GetAKTPrice() (float64, error) {
	cacheFile := "/tmp/aktprice.cache"
	price, err := readCachedPrice(cacheFile)
	if err == nil {
		return price, nil
	}

	price, err = fetchPriceFromAPI()
	if err != nil {
		return 0, err
	}

	if err := cachePrice(cacheFile, price); err != nil {
		return 0, err
	}

	return price, nil
}

// readCachedPrice reads the AKT price from the cache file.
func readCachedPrice(cacheFile string) (float64, error) {
	fileInfo, err := os.Stat(cacheFile)
	if os.IsNotExist(err) || time.Since(fileInfo.ModTime()) > 60*time.Minute {
		return 0, fmt.Errorf("cache file does not exist or is expired")
	}

	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(string(data), 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

// fetchPriceFromAPI tries to fetch the AKT price from primary and fallback APIs.
func fetchPriceFromAPI() (float64, error) {
	primaryURL := "https://api-osmosis.imperator.co/tokens/v2/price/AKT"
	fallbackURL := "https://api.coingecko.com/api/v3/simple/price?ids=akash-network&vs_currencies=usd"

	price, err := fetchPriceFromURL(primaryURL)
	if err != nil {
		fmt.Println("Primary API failed, trying fallback")
		return fetchPriceFromURL(fallbackURL)
	}

	return price, nil
}

// fetchPriceFromURL fetches the AKT price from a given URL.
func fetchPriceFromURL(url string) (float64, error) {
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var data interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}

	return extractPrice(data), nil
}

// extractPrice extracts the AKT price from the API response.
func extractPrice(data interface{}) float64 {
	switch v := data.(type) {
	case map[string]interface{}:
		if price, ok := v["price"].(float64); ok {
			return price
		}
		if nested, ok := v["akash-network"].(map[string]interface{}); ok {
			if price, ok := nested["usd"].(float64); ok {
				return price
			}
		}
	}
	return 0
}

// cachePrice writes the AKT price to the cache file.
func cachePrice(cacheFile string, price float64) error {
	return ioutil.WriteFile(cacheFile, []byte(fmt.Sprintf("%f", price)), 0644)
}
