package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/hashicorp/go-hclog"
)

type BucketResource struct {
	Name         string `json:"name"`
	Arn          string `json:"arn"`
	Region       string `json:"region"`
	CreationDate string `json:"creation_date,omitempty"`
}

type BucketDataset struct {
	Bucket  BucketResource      `json:"bucket"`
	Context BucketPolicyContext `json:"bucket_context"`
}

type BucketPolicyContext struct {
	Current          BucketCurrent             `json:"current"`
	BucketHomeRegion string                    `json:"bucket_home_region"`
	Tags             map[string]string         `json:"tags,omitempty"`
	Encryption       BucketEncryption          `json:"encryption,omitempty"`
	PublicAccessBlock BucketPublicAccessBlock  `json:"public_access_block,omitempty"`
	Policy           BucketPolicyDocument      `json:"policy,omitempty"`
	PolicyHash       string                    `json:"policy_hash,omitempty"`
	PolicyStatus     BucketPolicyStatus        `json:"policy_status,omitempty"`
	OwnershipControls BucketOwnershipControls  `json:"ownership_controls,omitempty"`
	Versioning       BucketVersioning          `json:"versioning,omitempty"`
	ObjectLock       BucketObjectLock          `json:"object_lock,omitempty"`
	Lifecycle        BucketLifecycle           `json:"lifecycle,omitempty"`
	Replication      BucketReplication         `json:"replication,omitempty"`
	Logging          BucketLogging             `json:"logging,omitempty"`
	Website          BucketWebsite             `json:"website,omitempty"`
}

type BucketCurrent struct {
	BucketName                    string `json:"bucket_name"`
	BucketArn                     string `json:"bucket_arn"`
	Region                        string `json:"region"`
	TagsPresent                   bool   `json:"tags_present"`
	IsPublic                      bool   `json:"is_public"`
	EncryptionEnabled             bool   `json:"encryption_enabled"`
	UsesKms                       bool   `json:"uses_kms"`
	OwnershipEnforced             bool   `json:"ownership_enforced"`
	IsVersioned                   bool   `json:"is_versioned"`
	MfaDeleteEnabled              bool   `json:"mfa_delete_enabled"`
	ObjectLockEnabled             bool   `json:"object_lock_enabled"`
	HasLifecycleRules             bool   `json:"has_lifecycle_rules"`
	LifecycleMinExpirationDays    int    `json:"lifecycle_min_expiration_days"`
	HasNoncurrentVersionExpiration bool  `json:"has_noncurrent_version_expiration"`
	LoggingEnabled                bool   `json:"logging_enabled"`
	ReplicationEnabled            bool   `json:"replication_enabled"`
	WebsiteHostingEnabled         bool   `json:"website_hosting_enabled"`
}

type BucketEncryption struct {
	Rules []BucketEncryptionRule `json:"rules,omitempty"`
}

type BucketEncryptionRule struct {
	Algorithm        string `json:"algorithm,omitempty"`
	KmsKeyArn        string `json:"kms_key_arn,omitempty"`
	BucketKeyEnabled bool   `json:"bucket_key_enabled"`
}

type BucketPublicAccessBlock struct {
	BlockPublicAcls       bool `json:"block_public_acls"`
	IgnorePublicAcls      bool `json:"ignore_public_acls"`
	BlockPublicPolicy     bool `json:"block_public_policy"`
	RestrictPublicBuckets bool `json:"restrict_public_buckets"`
}

type BucketPolicyDocument struct {
	Raw string `json:"raw,omitempty"`
}

type BucketPolicyStatus struct {
	IsPublic bool `json:"is_public"`
}

type BucketOwnershipControls struct {
	ObjectOwnership string `json:"object_ownership,omitempty"`
}

type BucketVersioning struct {
	Status    string `json:"status,omitempty"`
	MFADelete string `json:"mfa_delete,omitempty"`
}

type BucketObjectLock struct {
	Enabled              bool   `json:"enabled"`
	DefaultRetentionMode string `json:"default_retention_mode,omitempty"`
	DefaultRetentionDays int    `json:"default_retention_days,omitempty"`
	DefaultRetentionYears int   `json:"default_retention_years,omitempty"`
}

type BucketLifecycle struct {
	Rules []BucketLifecycleRule `json:"rules,omitempty"`
}

type BucketLifecycleRule struct {
	ID                                      string `json:"id,omitempty"`
	Status                                  string `json:"status,omitempty"`
	ExpirationDays                          int    `json:"expiration_days,omitempty"`
	NoncurrentVersionExpirationDays         int    `json:"noncurrent_version_expiration_days,omitempty"`
	AbortIncompleteMultipartUploadAfterDays int    `json:"abort_incomplete_multipart_upload_after_days,omitempty"`
	ExpiredObjectDeleteMarker               bool   `json:"expired_object_delete_marker"`
}

type BucketReplication struct {
	Role  string                  `json:"role,omitempty"`
	Rules []BucketReplicationRule `json:"rules,omitempty"`
}

type BucketReplicationRule struct {
	ID                   string `json:"id,omitempty"`
	Status               string `json:"status,omitempty"`
	DestinationBucketArn string `json:"destination_bucket_arn,omitempty"`
}

type BucketLogging struct {
	TargetBucket string `json:"target_bucket,omitempty"`
	TargetPrefix string `json:"target_prefix,omitempty"`
}

type BucketWebsite struct {
	HostingEnabled bool   `json:"hosting_enabled"`
	IndexSuffix    string `json:"index_suffix,omitempty"`
}

func CollectBucketDatasets(ctx context.Context, logger hclog.Logger, cfg aws.Config, allowedRegions []string) ([]BucketDataset, error) {
	client := s3.NewFromConfig(cfg)
	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	clientsByRegion := map[string]*s3.Client{cfg.Region: client}
	allowed := make(map[string]bool)
	for _, region := range allowedRegions {
		allowed[region] = true
	}

	datasets := make([]BucketDataset, 0, len(output.Buckets))
	var accumulatedErrors error
	for _, bucket := range output.Buckets {
		bucketName := aws.ToString(bucket.Name)
		if bucketName == "" {
			continue
		}

		bucketRegion, err := getBucketRegion(ctx, client, bucketName)
		if err != nil {
			logger.Error("unable to resolve bucket region", "bucket_name", bucketName, "error", err)
			accumulatedErrors = errors.Join(accumulatedErrors, err)
			continue
		}
		if len(allowed) > 0 && !allowed[bucketRegion] {
			continue
		}

		regionClient := clientsByRegion[bucketRegion]
		if regionClient == nil {
			regionCfg := cfg
			regionCfg.Region = bucketRegion
			regionClient = s3.NewFromConfig(regionCfg)
			clientsByRegion[bucketRegion] = regionClient
		}

		dataset, err := collectBucketDataset(ctx, regionClient, bucketName, bucketRegion, aws.ToTime(bucket.CreationDate))
		if err != nil {
			logger.Error("unable to collect bucket dataset", "bucket_name", bucketName, "region", bucketRegion, "error", err)
			accumulatedErrors = errors.Join(accumulatedErrors, err)
			continue
		}
		datasets = append(datasets, dataset)
	}

	return datasets, accumulatedErrors
}

func BuildBucketPolicyInput(bucket BucketDataset) (map[string]interface{}, error) {
	bucketValue, err := toInterfaceMap(bucket.Bucket)
	if err != nil {
		return nil, err
	}

	contextValue, err := toInterfaceMap(bucket.Context)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"bucket":         bucketValue,
		"bucket_context": contextValue,
	}, nil
}

func BucketDisplayName(bucket BucketDataset) string {
	return bucket.Bucket.Name
}

func collectBucketDataset(ctx context.Context, client *s3.Client, bucketName string, region string, creationDate time.Time) (BucketDataset, error) {
	tags, err := getBucketTags(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	encryption, err := getBucketEncryption(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	publicAccessBlock, err := getBucketPublicAccessBlock(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	policy, policyHash, err := getBucketPolicy(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	policyStatus, err := getBucketPolicyStatus(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	ownershipControls, err := getBucketOwnershipControls(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	versioning, err := getBucketVersioning(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	objectLock, err := getBucketObjectLock(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	lifecycle, err := getBucketLifecycle(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	replication, err := getBucketReplication(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	logging, err := getBucketLogging(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}
	website, err := getBucketWebsite(ctx, client, bucketName)
	if err != nil {
		return BucketDataset{}, err
	}

	bucketArn := fmt.Sprintf("arn:aws:s3:::%s", bucketName)
	current := BucketCurrent{
		BucketName:                    bucketName,
		BucketArn:                     bucketArn,
		Region:                        region,
		TagsPresent:                   len(tags) > 0,
		IsPublic:                      policyStatus.IsPublic,
		EncryptionEnabled:             len(encryption.Rules) > 0,
		UsesKms:                       bucketUsesKMS(encryption),
		OwnershipEnforced:             ownershipControls.ObjectOwnership == string(s3types.ObjectOwnershipBucketOwnerEnforced),
		IsVersioned:                   versioning.Status == string(s3types.BucketVersioningStatusEnabled),
		MfaDeleteEnabled:              versioning.MFADelete == string(s3types.MFADeleteStatusEnabled),
		ObjectLockEnabled:             objectLock.Enabled,
		HasLifecycleRules:             len(lifecycle.Rules) > 0,
		LifecycleMinExpirationDays:    minLifecycleExpirationDays(lifecycle),
		HasNoncurrentVersionExpiration: hasNoncurrentVersionExpiration(lifecycle),
		LoggingEnabled:                strings.TrimSpace(logging.TargetBucket) != "",
		ReplicationEnabled:            hasEnabledReplication(replication),
		WebsiteHostingEnabled:         website.HostingEnabled,
	}

	return BucketDataset{
		Bucket: BucketResource{
			Name:         bucketName,
			Arn:          bucketArn,
			Region:       region,
			CreationDate: formatTime(creationDate),
		},
		Context: BucketPolicyContext{
			Current:           current,
			BucketHomeRegion:  region,
			Tags:              tags,
			Encryption:        encryption,
			PublicAccessBlock: publicAccessBlock,
			Policy:            policy,
			PolicyHash:        policyHash,
			PolicyStatus:      policyStatus,
			OwnershipControls: ownershipControls,
			Versioning:        versioning,
			ObjectLock:        objectLock,
			Lifecycle:         lifecycle,
			Replication:       replication,
			Logging:           logging,
			Website:           website,
		},
	}, nil
}

func toInterfaceMap(value interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoded := make(map[string]interface{})
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func getBucketRegion(ctx context.Context, client *s3.Client, bucketName string) (string, error) {
	output, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: aws.String(bucketName)})
	if err != nil {
		return "", err
	}
	return normalizeBucketRegion(output.LocationConstraint), nil
}

func normalizeBucketRegion(location s3types.BucketLocationConstraint) string {
	region := string(location)
	if region == "" {
		return "us-east-1"
	}
	if region == "EU" {
		return "eu-west-1"
	}
	return region
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func bucketUsesKMS(encryption BucketEncryption) bool {
	for _, rule := range encryption.Rules {
		if rule.Algorithm == string(s3types.ServerSideEncryptionAwsKms) {
			return true
		}
	}
	return false
}

func minLifecycleExpirationDays(lifecycle BucketLifecycle) int {
	minDays := 0
	for _, rule := range lifecycle.Rules {
		if rule.ExpirationDays <= 0 {
			continue
		}
		if minDays == 0 || rule.ExpirationDays < minDays {
			minDays = rule.ExpirationDays
		}
	}
	return minDays
}

func hasNoncurrentVersionExpiration(lifecycle BucketLifecycle) bool {
	for _, rule := range lifecycle.Rules {
		if rule.NoncurrentVersionExpirationDays > 0 {
			return true
		}
	}
	return false
}

func hasEnabledReplication(replication BucketReplication) bool {
	for _, rule := range replication.Rules {
		if strings.EqualFold(rule.Status, string(s3types.ReplicationRuleStatusEnabled)) {
			return true
		}
	}
	return false
}

func isAPIErrorCode(err error, codes ...string) bool {
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	for _, code := range codes {
		if apiErr.ErrorCode() == code {
			return true
		}
	}
	return false
}
