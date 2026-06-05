package main

import "testing"

func TestSupportedPolicyBehaviors(t *testing.T) {
	behaviors := supportedPolicyBehaviors()
	if len(behaviors) != 1 || behaviors[0] != "bucket" {
		t.Fatalf("supportedPolicyBehaviors() = %v, want [bucket]", behaviors)
	}
}

func TestBuildRequiredDatasetsForBucketPolicies(t *testing.T) {
	required := buildRequiredDatasets(map[string][]string{
		"bucket": {"/tmp/bundle.tar.gz"},
	})
	if !required["buckets"] {
		t.Fatalf("buildRequiredDatasets() = %v, want buckets dataset to be required", required)
	}
}
