package main

import (
	"testing"
	"time"
)

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

func TestNewEvalContextHasDefaultTimeout(t *testing.T) {
	ctx, cancel := newEvalContext()
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("newEvalContext() deadline missing")
	}

	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > defaultEvalTimeout {
		t.Fatalf("newEvalContext() deadline remaining = %s, want within %s", remaining, defaultEvalTimeout)
	}
}
