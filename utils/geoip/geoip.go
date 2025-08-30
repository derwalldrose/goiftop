package geoip

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/fs714/goiftop/utils/log"
)

type GeoInfo struct {
	Country string `json:"country"`
	City    string `json:"city"`
}

var (
	geoCache = make(map[string]GeoInfo)
	cacheMux = &sync.RWMutex{}
)

const (
	apiURL = "http://ip-api.com/json/"
)

// Lookup performs a lookup for the given IP address.
// It first checks the cache, and if not found, queries the external API.
func Lookup(ipStr string) (GeoInfo, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil || ip.IsPrivate() || ip.IsLoopback() {
		return GeoInfo{}, fmt.Errorf("invalid or private IP")
	}

	cacheMux.RLock()
	info, found := geoCache[ipStr]
	cacheMux.RUnlock()

	if found {
		return info, nil
	}

	return lookupFromAPI(ipStr)
}

func lookupFromAPI(ipStr string) (GeoInfo, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	url := fmt.Sprintf("%s%s?fields=country,city", apiURL, ipStr)

	resp, err := client.Get(url)
	if err != nil {
		return GeoInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GeoInfo{}, fmt.Errorf("api request failed with status: %s", resp.Status)
	}

	var apiResponse struct {
		Status  string `json:"status"`
		Country string `json:"country"`
		City    string `json:"city"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return GeoInfo{}, err
	}

	if apiResponse.Status == "fail" {
		return GeoInfo{}, fmt.Errorf("api error: %s", apiResponse.Message)
	}

	info := GeoInfo{
		Country: apiResponse.Country,
		City:    apiResponse.City,
	}

	cacheMux.Lock()
	geoCache[ipStr] = info
	cacheMux.Unlock()

	log.Infof("geoip lookup for %s, result: %s, %s", ipStr, info.Country, info.City)

	return info, nil
}
