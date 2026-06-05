package internal

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func getBucketTags(ctx context.Context, client *s3.Client, bucketName string) (map[string]string, error) {
	output, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "NoSuchTagSet") {
			return map[string]string{}, nil
		}
		return nil, err
	}

	tags := make(map[string]string, len(output.TagSet))
	for _, tag := range output.TagSet {
		key := aws.ToString(tag.Key)
		if key == "" {
			continue
		}
		tags[key] = aws.ToString(tag.Value)
	}
	return tags, nil
}

func getBucketEncryption(ctx context.Context, client *s3.Client, bucketName string) (BucketEncryption, error) {
	output, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "ServerSideEncryptionConfigurationNotFoundError") {
			return BucketEncryption{}, nil
		}
		return BucketEncryption{}, err
	}

	return bucketEncryptionFromOutput(output), nil
}

func bucketEncryptionFromOutput(output *s3.GetBucketEncryptionOutput) BucketEncryption {
	if output == nil || output.ServerSideEncryptionConfiguration == nil {
		return BucketEncryption{}
	}

	rules := make([]BucketEncryptionRule, 0, len(output.ServerSideEncryptionConfiguration.Rules))
	for _, rule := range output.ServerSideEncryptionConfiguration.Rules {
		if rule.ApplyServerSideEncryptionByDefault == nil {
			continue
		}
		rules = append(rules, BucketEncryptionRule{
			Algorithm:        string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm),
			KmsKeyArn:        aws.ToString(rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID),
			BucketKeyEnabled: aws.ToBool(rule.BucketKeyEnabled),
		})
	}
	return BucketEncryption{Rules: rules}
}

func getBucketPublicAccessBlock(ctx context.Context, client *s3.Client, bucketName string) (BucketPublicAccessBlock, error) {
	output, err := client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "NoSuchPublicAccessBlockConfiguration") {
			return BucketPublicAccessBlock{}, nil
		}
		return BucketPublicAccessBlock{}, err
	}

	return bucketPublicAccessBlockFromOutput(output), nil
}

func bucketPublicAccessBlockFromOutput(output *s3.GetPublicAccessBlockOutput) BucketPublicAccessBlock {
	if output == nil || output.PublicAccessBlockConfiguration == nil {
		return BucketPublicAccessBlock{}
	}

	pab := output.PublicAccessBlockConfiguration
	return BucketPublicAccessBlock{
		BlockPublicAcls:       aws.ToBool(pab.BlockPublicAcls),
		IgnorePublicAcls:      aws.ToBool(pab.IgnorePublicAcls),
		BlockPublicPolicy:     aws.ToBool(pab.BlockPublicPolicy),
		RestrictPublicBuckets: aws.ToBool(pab.RestrictPublicBuckets),
	}
}

func getBucketPolicy(ctx context.Context, client *s3.Client, bucketName string) (BucketPolicyDocument, string, error) {
	output, err := client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "NoSuchBucketPolicy") {
			return BucketPolicyDocument{}, "", nil
		}
		return BucketPolicyDocument{}, "", err
	}
	policy := aws.ToString(output.Policy)
	return BucketPolicyDocument{Raw: policy}, SHA256Hex(policy), nil
}

func getBucketPolicyStatus(ctx context.Context, client *s3.Client, bucketName string) (BucketPolicyStatus, error) {
	output, err := client.GetBucketPolicyStatus(ctx, &s3.GetBucketPolicyStatusInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "NoSuchBucketPolicy") {
			return BucketPolicyStatus{}, nil
		}
		return BucketPolicyStatus{}, err
	}

	return bucketPolicyStatusFromOutput(output), nil
}

func bucketPolicyStatusFromOutput(output *s3.GetBucketPolicyStatusOutput) BucketPolicyStatus {
	if output == nil || output.PolicyStatus == nil {
		return BucketPolicyStatus{}
	}

	return BucketPolicyStatus{IsPublic: aws.ToBool(output.PolicyStatus.IsPublic)}
}

func getBucketOwnershipControls(ctx context.Context, client *s3.Client, bucketName string) (BucketOwnershipControls, error) {
	output, err := client.GetBucketOwnershipControls(ctx, &s3.GetBucketOwnershipControlsInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "OwnershipControlsNotFoundError") {
			return BucketOwnershipControls{}, nil
		}
		return BucketOwnershipControls{}, err
	}
	if output.OwnershipControls == nil || len(output.OwnershipControls.Rules) == 0 {
		return BucketOwnershipControls{}, nil
	}
	return BucketOwnershipControls{ObjectOwnership: string(output.OwnershipControls.Rules[0].ObjectOwnership)}, nil
}

func getBucketVersioning(ctx context.Context, client *s3.Client, bucketName string) (BucketVersioning, error) {
	output, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: aws.String(bucketName)})
	if err != nil {
		return BucketVersioning{}, err
	}
	return BucketVersioning{
		Status:    string(output.Status),
		MFADelete: string(output.MFADelete),
	}, nil
}

func getBucketObjectLock(ctx context.Context, client *s3.Client, bucketName string) (BucketObjectLock, error) {
	output, err := client.GetObjectLockConfiguration(ctx, &s3.GetObjectLockConfigurationInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "ObjectLockConfigurationNotFoundError") {
			return BucketObjectLock{}, nil
		}
		return BucketObjectLock{}, err
	}
	if output.ObjectLockConfiguration == nil {
		return BucketObjectLock{}, nil
	}

	result := BucketObjectLock{
		Enabled: output.ObjectLockConfiguration.ObjectLockEnabled == s3types.ObjectLockEnabledEnabled,
	}
	if output.ObjectLockConfiguration.Rule != nil && output.ObjectLockConfiguration.Rule.DefaultRetention != nil {
		retention := output.ObjectLockConfiguration.Rule.DefaultRetention
		result.DefaultRetentionMode = string(retention.Mode)
		result.DefaultRetentionDays = int(aws.ToInt32(retention.Days))
		result.DefaultRetentionYears = int(aws.ToInt32(retention.Years))
	}
	return result, nil
}

func getBucketLifecycle(ctx context.Context, client *s3.Client, bucketName string) (BucketLifecycle, error) {
	output, err := client.GetBucketLifecycleConfiguration(ctx, &s3.GetBucketLifecycleConfigurationInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "NoSuchLifecycleConfiguration") {
			return BucketLifecycle{}, nil
		}
		return BucketLifecycle{}, err
	}

	rules := make([]BucketLifecycleRule, 0, len(output.Rules))
	for _, rule := range output.Rules {
		entry := BucketLifecycleRule{
			ID:     aws.ToString(rule.ID),
			Status: string(rule.Status),
		}
		if rule.Expiration != nil {
			entry.ExpirationDays = int(aws.ToInt32(rule.Expiration.Days))
			entry.ExpiredObjectDeleteMarker = aws.ToBool(rule.Expiration.ExpiredObjectDeleteMarker)
		}
		if rule.NoncurrentVersionExpiration != nil {
			entry.NoncurrentVersionExpirationDays = int(aws.ToInt32(rule.NoncurrentVersionExpiration.NoncurrentDays))
		}
		if rule.AbortIncompleteMultipartUpload != nil {
			entry.AbortIncompleteMultipartUploadAfterDays = int(aws.ToInt32(rule.AbortIncompleteMultipartUpload.DaysAfterInitiation))
		}
		rules = append(rules, entry)
	}
	return BucketLifecycle{Rules: rules}, nil
}

func getBucketReplication(ctx context.Context, client *s3.Client, bucketName string) (BucketReplication, error) {
	output, err := client.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "ReplicationConfigurationNotFoundError") {
			return BucketReplication{}, nil
		}
		return BucketReplication{}, err
	}

	return bucketReplicationFromOutput(output), nil
}

func bucketReplicationFromOutput(output *s3.GetBucketReplicationOutput) BucketReplication {
	if output == nil || output.ReplicationConfiguration == nil {
		return BucketReplication{}
	}

	rules := make([]BucketReplicationRule, 0, len(output.ReplicationConfiguration.Rules))
	for _, rule := range output.ReplicationConfiguration.Rules {
		if rule.Destination == nil {
			continue
		}
		rules = append(rules, BucketReplicationRule{
			ID:                   aws.ToString(rule.ID),
			Status:               string(rule.Status),
			DestinationBucketArn: aws.ToString(rule.Destination.Bucket),
		})
	}
	return BucketReplication{Role: aws.ToString(output.ReplicationConfiguration.Role), Rules: rules}
}

func getBucketLogging(ctx context.Context, client *s3.Client, bucketName string) (BucketLogging, error) {
	output, err := client.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{Bucket: aws.String(bucketName)})
	if err != nil {
		return BucketLogging{}, err
	}
	if output.LoggingEnabled == nil {
		return BucketLogging{}, nil
	}
	return BucketLogging{
		TargetBucket: aws.ToString(output.LoggingEnabled.TargetBucket),
		TargetPrefix: aws.ToString(output.LoggingEnabled.TargetPrefix),
	}, nil
}

func getBucketWebsite(ctx context.Context, client *s3.Client, bucketName string) (BucketWebsite, error) {
	output, err := client.GetBucketWebsite(ctx, &s3.GetBucketWebsiteInput{Bucket: aws.String(bucketName)})
	if err != nil {
		if isAPIErrorCode(err, "NoSuchWebsiteConfiguration") {
			return BucketWebsite{}, nil
		}
		return BucketWebsite{}, err
	}
	website := BucketWebsite{HostingEnabled: true}
	if output.IndexDocument != nil {
		website.IndexSuffix = aws.ToString(output.IndexDocument.Suffix)
	}
	return website, nil
}
