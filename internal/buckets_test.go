package internal

import "testing"

func TestBuildBucketPolicyInputIncludesBucketAndContext(t *testing.T) {
	input, err := BuildBucketPolicyInput(BucketDataset{
		Bucket: BucketResource{
			Name:   "example-bucket",
			Arn:    "arn:aws:s3:::example-bucket",
			Region: "eu-west-2",
		},
		Context: BucketPolicyContext{
			Current: BucketCurrent{
				BucketName:         "example-bucket",
				BucketArn:          "arn:aws:s3:::example-bucket",
				Region:             "eu-west-2",
				EncryptionEnabled:  true,
				IsVersioned:        true,
				HasLifecycleRules:  true,
				LoggingEnabled:     true,
				ReplicationEnabled: true,
			},
			Tags: map[string]string{"evidence": "true"},
		},
	})
	if err != nil {
		t.Fatalf("BuildBucketPolicyInput() error = %v", err)
	}
	if _, ok := input["bucket"]; !ok {
		t.Fatalf("BuildBucketPolicyInput() missing bucket key: %v", input)
	}
	if _, ok := input["bucket_context"]; !ok {
		t.Fatalf("BuildBucketPolicyInput() missing bucket_context key: %v", input)
	}
}
