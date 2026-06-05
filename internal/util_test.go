package internal

import "testing"

func TestResolveRegionsIgnoresWhitespaceOnlyRegionConfig(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-2")

	regions := ResolveRegions(map[string]string{"region": "   \t  "})
	if len(regions) != 1 || regions[0] != "eu-west-2" {
		t.Fatalf("ResolveRegions() = %v, want [eu-west-2]", regions)
	}
}

func TestSHA256HexIsStable(t *testing.T) {
	first := SHA256Hex("bucket-policy")
	second := SHA256Hex("bucket-policy")
	if first == "" || first != second {
		t.Fatalf("SHA256Hex() produced unstable output: %q vs %q", first, second)
	}
}
