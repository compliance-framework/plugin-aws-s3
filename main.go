package main

import (
	"context"
	"errors"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/compliance-framework/agent/runner"
	"github.com/compliance-framework/agent/runner/proto"
	"github.com/compliance-framework/plugin-aws-s3/internal"
	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
)

type CompliancePlugin struct {
	logger     hclog.Logger
	rawConfig  map[string]string
	config     *internal.PluginConfig
	policyData map[string]interface{}
}

func (l *CompliancePlugin) Configure(req *proto.ConfigureRequest) (*proto.ConfigureResponse, error) {

	rawConfig := req.GetConfig()
	parsedConfig, err := internal.ParseConfig(rawConfig)
	if err != nil {
		return nil, err
	}
	l.rawConfig = rawConfig
	l.config = parsedConfig

	// Maps policy data for policy data.xx evaluation. This lets the agent set configuration for policy
	// execution. Uses are for overriding values like set list of tags, approved names etc.
	if req.GetPolicyData() != nil {
		l.policyData = req.GetPolicyData().AsMap()
	} else {
		l.policyData = nil
	}

	return &proto.ConfigureResponse{}, nil
}

func (l *CompliancePlugin) Init(req *proto.InitRequest, apiHelper runner.ApiHelper) (*proto.InitResponse, error) {
	ctx := context.Background()
	return runner.InitWithSubjectsAndRisksFromPolicies(ctx, l.logger, req, apiHelper, buildSubjectTemplates())
}

func (l *CompliancePlugin) Eval(request *proto.EvalRequest, apiHelper runner.ApiHelper) (*proto.EvalResponse, error) {
	ctx := context.Background()
	evalStatus := proto.ExecutionStatus_SUCCESS
	var accumulatedErrors error

	if request == nil {
		return &proto.EvalResponse{Status: proto.ExecutionStatus_FAILURE}, fmt.Errorf("eval request is nil")
	}

	regions := internal.ResolveRegions(l.rawConfig)
	actors := []*proto.OriginActor{
		{
			Title: "The Continuous Compliance Framework",
			Type:  "assessment-platform",
			Links: []*proto.Link{
				{
					Href: "https://compliance-framework.github.io/docs/",
					Rel:  internal.StringAddressed("reference"),
					Text: internal.StringAddressed("The Continuous Compliance Framework"),
				},
			},
		},
		{
			Title: "Continuous Compliance Framework - AWS S3 Plugin",
			Type:  "tool",
			Links: []*proto.Link{
				{
					Href: "https://github.com/compliance-framework/plugin-aws-s3",
					Rel:  internal.StringAddressed("reference"),
					Text: internal.StringAddressed("The Continuous Compliance Framework AWS S3 Plugin"),
				},
			},
		},
	}

	defaultBehaviorMapping := map[string][]string{
		"aws-s3-policies": {"bucket"},
	}
	policyEval := request.WithDefaultPolicyBehavior(defaultBehaviorMapping)
	policyPathsByBehavior := buildPolicyPathsByBehavior(policyEval)
	requiredDatasets := buildRequiredDatasets(policyPathsByBehavior)

	if !requiredDatasets["buckets"] {
		return &proto.EvalResponse{Status: evalStatus}, nil
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(regions[0]))
	if err != nil {
		return &proto.EvalResponse{
			Status: proto.ExecutionStatus_FAILURE,
		}, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	buckets, err := internal.CollectBucketDatasets(ctx, l.logger, cfg, regions)
	if err != nil {
		evalStatus = proto.ExecutionStatus_FAILURE
		accumulatedErrors = errors.Join(accumulatedErrors, err)
	}

	deps := internal.EvaluationDependencies{
		Context:    ctx,
		Logger:     l.logger,
		ApiHelper:  apiHelper,
		Actors:     actors,
		PolicyData: l.policyData,
	}

	if bucketPolicyPaths := policyPathsByBehavior["bucket"]; len(bucketPolicyPaths) > 0 {
		result := internal.EvaluateBucketPolicies(deps, bucketPolicyPaths, buckets)
		if fatal := applyResourceEvaluationErrors(result, &evalStatus, &accumulatedErrors, true); fatal != nil {
			return &proto.EvalResponse{Status: proto.ExecutionStatus_FAILURE}, fatal
		}
	}

	return &proto.EvalResponse{Status: evalStatus}, accumulatedErrors
}

func applyResourceEvaluationErrors(result internal.ResourceEvaluationErrors, evalStatus *proto.ExecutionStatus, accumulatedErrors *error, failOnInputBuild bool) error {
	if result.NonFatal != nil {
		*accumulatedErrors = errors.Join(*accumulatedErrors, result.NonFatal)
	}
	if failOnInputBuild && result.InputBuildFailure {
		*evalStatus = proto.ExecutionStatus_FAILURE
	}
	return result.Fatal
}

func supportedPolicyBehaviors() []string {
	return []string{"bucket"}
}

func buildPolicyPathsByBehavior(request *proto.EvalRequest) map[string][]string {
	policyPathsByBehavior := make(map[string][]string)
	for _, behavior := range supportedPolicyBehaviors() {
		policyPaths := request.PolicyPathsForBehavior(behavior)
		if len(policyPaths) > 0 {
			policyPathsByBehavior[behavior] = policyPaths
		}
	}
	return policyPathsByBehavior
}

func buildRequiredDatasets(policyPathsByBehavior map[string][]string) map[string]bool {
	requiredDatasets := make(map[string]bool)
	for behavior, policyPaths := range policyPathsByBehavior {
		if len(policyPaths) == 0 {
			continue
		}

		switch behavior {
		case "bucket":
			markRequiredDatasets(requiredDatasets, "buckets")
		}
	}
	return requiredDatasets
}

func markRequiredDatasets(requiredDatasets map[string]bool, datasetNames ...string) {
	for _, datasetName := range datasetNames {
		requiredDatasets[datasetName] = true
	}
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Debug,
		JSONFormat: true,
	})

	compliancePluginObj := &CompliancePlugin{
		logger: logger,
	}
	// pluginMap is the map of plugins we can dispense.
	logger.Debug("Initiating AWS S3 plugin")

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: runner.HandshakeConfig,
		Plugins: map[string]goplugin.Plugin{
			"runner": &runner.RunnerV2GRPCPlugin{
				Impl: compliancePluginObj,
			},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
