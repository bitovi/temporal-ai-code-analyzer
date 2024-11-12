package utils

import (
	"net/http"
	"os"

	httputils "bitovi.com/code-analyzer/src/utils/http"
)

var ChaosServerURL = os.Getenv("CHAOS_SERVER_URL")

func ChaosExists(key string) bool {
	chaosResponse, err := httputils.GetRequest(ChaosServerURL + "?key=" + key)
	if err != nil {
		return true
	}
	defer chaosResponse.Body.Close()
	return chaosResponse.StatusCode != http.StatusOK
}
