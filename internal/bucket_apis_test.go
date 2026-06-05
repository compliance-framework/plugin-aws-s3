package internal

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestBucketEncryptionFromOutputHandlesNilConfiguration(t *testing.T) {
	encryption := bucketEncryptionFromOutput(&s3.GetBucketEncryptionOutput{})
	if len(encryption.Rules) != 0 {
		t.Fatalf("bucketEncryptionFromOutput() = %v, want no rules", encryption)
	}
}

func TestBucketEncryptionFromOutputSkipsIncompleteRules(t *testing.T) {
	encryption := bucketEncryptionFromOutput(&s3.GetBucketEncryptionOutput{
		ServerSideEncryptionConfiguration: &s3types.ServerSideEncryptionConfiguration{
			Rules: []s3types.ServerSideEncryptionRule{
				{},
				{
					ApplyServerSideEncryptionByDefault: &s3types.ServerSideEncryptionByDefault{
						SSEAlgorithm:   s3types.ServerSideEncryptionAwsKms,
						KMSMasterKeyID: aws.String("arn:aws:kms:eu-west-2:123456789012:key/example"),
					},
					BucketKeyEnabled: aws.Bool(true),
				},
			},
		},
	})

	if len(encryption.Rules) != 1 {
		t.Fatalf("bucketEncryptionFromOutput() rules = %v, want one complete rule", encryption.Rules)
	}
	if encryption.Rules[0].Algorithm != string(s3types.ServerSideEncryptionAwsKms) {
		t.Fatalf("bucketEncryptionFromOutput() algorithm = %q, want %q", encryption.Rules[0].Algorithm, s3types.ServerSideEncryptionAwsKms)
	}
}

func TestBucketPublicAccessBlockFromOutputHandlesNilConfiguration(t *testing.T) {
	pab := bucketPublicAccessBlockFromOutput(&s3.GetPublicAccessBlockOutput{})
	if pab != (BucketPublicAccessBlock{}) {
		t.Fatalf("bucketPublicAccessBlockFromOutput() = %v, want zero value", pab)
	}
}

func TestBucketPolicyStatusFromOutputHandlesNilStatus(t *testing.T) {
	status := bucketPolicyStatusFromOutput(&s3.GetBucketPolicyStatusOutput{})
	if status != (BucketPolicyStatus{}) {
		t.Fatalf("bucketPolicyStatusFromOutput() = %v, want zero value", status)
	}
}

func TestBucketReplicationFromOutputHandlesNilConfiguration(t *testing.T) {
	replication := bucketReplicationFromOutput(&s3.GetBucketReplicationOutput{})
	if len(replication.Rules) != 0 || replication.Role != "" {
		t.Fatalf("bucketReplicationFromOutput() = %v, want zero value", replication)
	}
}

func TestBucketReplicationFromOutputSkipsIncompleteRules(t *testing.T) {
	replication := bucketReplicationFromOutput(&s3.GetBucketReplicationOutput{
		ReplicationConfiguration: &s3types.ReplicationConfiguration{
			Role: aws.String("arn:aws:iam::123456789012:role/s3-replication"),
			Rules: []s3types.ReplicationRule{
				{
					Status: s3types.ReplicationRuleStatusEnabled,
				},
				{
					Status: s3types.ReplicationRuleStatusEnabled,
					Destination: &s3types.Destination{
						Bucket: aws.String("arn:aws:s3:::destination-bucket"),
					},
				},
			},
		},
	})

	if len(replication.Rules) != 1 {
		t.Fatalf("bucketReplicationFromOutput() rules = %v, want one complete rule", replication.Rules)
	}
	if replication.Rules[0].DestinationBucketArn != "arn:aws:s3:::destination-bucket" {
		t.Fatalf("bucketReplicationFromOutput() destination = %q, want destination bucket ARN", replication.Rules[0].DestinationBucketArn)
	}
}
