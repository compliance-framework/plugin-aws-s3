package internal

import (
	"fmt"

	"github.com/compliance-framework/agent/runner/proto"
)

type BucketEvidenceContext struct {
	Labels     map[string]string
	Components []*proto.Component
	Inventory  []*proto.InventoryItem
	Subjects   []*proto.Subject
}

func BuildBucketEvidenceContext(bucket BucketResource) BucketEvidenceContext {
	labels := map[string]string{
		"provider":     "aws",
		"type":         "bucket",
		"bucket_name":  bucket.Name,
		"region":       bucket.Region,
		"resource_arn": bucket.Arn,
	}

	components := []*proto.Component{
		{
			Identifier:  "common-components/amazon-s3",
			Type:        "service",
			Title:       "Amazon S3",
			Description: "Amazon Simple Storage Service (S3) provides scalable object storage in AWS.",
			Purpose:     "To provide durable object storage evaluated for storage compliance posture.",
		},
	}

	inventoryID := fmt.Sprintf("aws-s3/%s/%s", bucket.Region, bucket.Name)
	inventory := []*proto.InventoryItem{
		{
			Identifier:  inventoryID,
			Type:        "object-storage",
			Title:       fmt.Sprintf("Amazon S3 bucket [%s]", bucket.Name),
			Description: fmt.Sprintf("Amazon S3 bucket %s in region %s.", bucket.Name, bucket.Region),
			Props: []*proto.Property{
				{Name: "bucket-name", Value: bucket.Name},
				{Name: "region", Value: bucket.Region},
				{Name: "resource-arn", Value: bucket.Arn},
			},
			ImplementedComponents: []*proto.InventoryItemImplementedComponent{
				{Identifier: "common-components/amazon-s3"},
			},
		},
	}

	subjects := []*proto.Subject{
		{
			Type:       proto.SubjectType_SUBJECT_TYPE_COMPONENT,
			Identifier: "common-components/amazon-s3",
		},
		{
			Type:       proto.SubjectType_SUBJECT_TYPE_INVENTORY_ITEM,
			Identifier: inventoryID,
		},
	}

	return BucketEvidenceContext{
		Labels:     labels,
		Components: components,
		Inventory:  inventory,
		Subjects:   subjects,
	}
}

func EvaluateBucketPolicies(deps EvaluationDependencies, policyPaths []string, buckets []BucketDataset) ResourceEvaluationErrors {
	return evaluateResources(
		deps,
		policyPaths,
		buckets,
		func(bucket BucketDataset) ResourceEvidenceContext {
			bucketCtx := BuildBucketEvidenceContext(bucket.Bucket)
			return newResourceEvidenceContext(bucketCtx.Labels, bucketCtx.Subjects, bucketCtx.Components, bucketCtx.Inventory)
		},
		func(bucket BucketDataset) (interface{}, error) {
			return BuildBucketPolicyInput(bucket)
		},
		func(bucket BucketDataset, err error) {
			deps.Logger.Error("unable to build bucket policy input", "bucket_name", bucket.Bucket.Name, "region", bucket.Bucket.Region, "error", err)
		},
		func(evidences []*proto.Evidence, bucket BucketDataset) {
			PrefixEvidenceTitles(evidences, BucketDisplayName(bucket))
		},
	)
}
