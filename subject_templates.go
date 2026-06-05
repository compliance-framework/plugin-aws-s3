package main

import "github.com/compliance-framework/agent/runner/proto"

func buildSubjectTemplates() []*proto.SubjectTemplate {
	return []*proto.SubjectTemplate{
		{
			Name:              "aws-s3-bucket",
			Type:              proto.SubjectType_SUBJECT_TYPE_COMPONENT,
			TitleTemplate:     `AWS S3 bucket {{ .bucket_name }} in {{ .region }}`,
			DescriptionTemplate: `Amazon S3 bucket {{ .bucket_name }} in AWS region {{ .region }}.`,
			PurposeTemplate:   "Represents an Amazon S3 bucket evaluated for storage compliance posture.",
			IdentityLabelKeys: []string{"provider", "region", "bucket_name"},
			SelectorLabels:    selectorLabelsForType("bucket"),
			LabelSchema: labelSchema(
				label("provider", "Cloud provider for the evaluated resource"),
				label("type", "S3 plugin resource type"),
				label("bucket_name", "Amazon S3 bucket name"),
				label("region", "AWS region containing the bucket"),
				label("resource_arn", "AWS resource ARN for the bucket"),
			),
		},
	}
}

func selectorLabelsForType(resourceType string) []*proto.SubjectLabelSelector {
	return []*proto.SubjectLabelSelector{
		{
			Key:   "type",
			Value: resourceType,
		},
	}
}

func label(key string, description string) *proto.SubjectLabelSchema {
	return &proto.SubjectLabelSchema{
		Key:         key,
		Description: description,
	}
}

func labelSchema(labels ...*proto.SubjectLabelSchema) []*proto.SubjectLabelSchema {
	return labels
}
