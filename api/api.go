package api

import (
	"encoding/base64"
	"os"
	"strings"

	"github.com/caleb-mwasikira/tap_gopay_backend/utils"
)

var (
	ANDROID_API_KEY string
)

func init() {
	utils.LoadDotenv()

	ANDROID_API_KEY = os.Getenv("ANDROID_API_KEY")
}

func GenerateAndroidApiKey() string {
	if strings.TrimSpace(ANDROID_API_KEY) == "" {
		return ""
	}
	b64EncodedData := base64.StdEncoding.EncodeToString([]byte(ANDROID_API_KEY))
	return b64EncodedData
}
