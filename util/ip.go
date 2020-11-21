package util

import (
	"encoding/json"
	"net/http"
)

const api = "https://api.ipify.org?format=json"

type apiResp struct {
	IP string `json:"ip"`
}

// GetPublicIP will query a public API to get the publicly accssible IP
func GetPublicIP() (string, error) {
	resp, err := http.Get(api)

	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var s apiResp
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", err
	}

	return s.IP, nil
}
