package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

func StringAddressed(str string) *string {
	return &str
}

func MergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, values := range maps {
		for k, v := range values {
			result[k] = v
		}
	}
	return result
}

func ResolveRegions(config map[string]string) []string {
	if regionsStr, ok := config["regions"]; ok && regionsStr != "" {
		regionParts := strings.Split(regionsStr, ",")
		regions := make([]string, 0, len(regionParts))
		seen := make(map[string]bool)
		for _, r := range regionParts {
			r = strings.TrimSpace(r)
			if r != "" && !seen[r] {
				seen[r] = true
				regions = append(regions, r)
			}
		}
		if len(regions) > 0 {
			return regions
		}
	}

	if regionStr, ok := config["region"]; ok {
		region := strings.TrimSpace(regionStr)
		if region != "" {
			return []string{region}
		}
	}

	if regionEnv := strings.TrimSpace(os.Getenv("AWS_REGION")); regionEnv != "" {
		return []string{regionEnv}
	}

	return []string{"us-east-1"}
}

func SHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
