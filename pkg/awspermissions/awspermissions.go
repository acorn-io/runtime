package awspermissions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/digest"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AWSAPIGroup          = "aws.acorn.io"
	AWSRoleAPIGroup      = "role.aws.acorn.io"
	AWSRoleHash          = "role.aws.acorn.io/hash"
	AssumeVerb           = "assume"
	AssumeRoleAnnotation = "eks.amazonaws.com/role-arn"
)

type policyDocument struct {
	Version   string
	Statement []statement
}

type statement struct {
	Effect    string
	Action    []string
	Resource  string         `json:",omitempty"`
	Condition map[string]any `json:",omitempty"`
	Principal map[string]any `json:",omitempty"`
}

func toPolicy(permissions v1.Permissions) (string, error) {
	var (
		policy = policyDocument{
			Version: "2012-10-17",
		}
	)

	for _, rule := range permissions.Rules {
		for _, apiGroup := range rule.APIGroups {
			if apiGroup != AWSAPIGroup {
				continue
			}

			if len(rule.Verbs) == 0 {
				continue
			}

			for _, resource := range rule.Resources {
				policy.Statement = append(policy.Statement, statement{
					Effect:   "Allow",
					Action:   rule.Verbs,
					Resource: resource,
				})
			}
		}
	}

	if len(policy.Statement) == 0 {
		return "", nil
	}

	data, err := json.Marshal(policy)
	return string(data), err
}

func getRoleARN(ctx context.Context, c kclient.Client, client *iam.Client, app *v1.AppInstance, identityProviderARN, serviceName, policy string) (string, error) {
	if created, err := alreadyCreated(ctx, c, app, serviceName, digest.SHA256(policy)); err != nil {
		return "", err
	} else if created != "" {
		return created, nil
	}

	roleName := name.SafeHashConcatName("acorn", app.Namespace, app.Name, serviceName)

	policyARN, err := createPolicy(ctx, client, app, serviceName, roleName, policy)
	if err != nil {
		return "", err
	}

	role, err := createRole(ctx, client, app, identityProviderARN, serviceName, roleName)
	if err != nil {
		return "", err
	}

	_, err = client.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		PolicyArn: &policyARN,
		RoleName:  role.RoleName,
	})
	if err != nil {
		return "", err
	}
	return *role.Arn, nil
}

func createRole(ctx context.Context, client *iam.Client, app *v1.AppInstance, identityProviderARN, serviceName, roleName string) (*types.Role, error) {
	var (
		found  *types.Role
		marker *string
	)

outer:
	for {
		roles, err := client.ListRoles(ctx, &iam.ListRolesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}

		for _, role := range roles.Roles {
			if role.RoleName != nil && *role.RoleName == roleName {
				found = &role
				break outer
			}
		}
		if roles.IsTruncated && roles.Marker != nil {
			marker = roles.Marker
			continue
		}
		break
	}

	if found != nil {
		_, err := client.DeleteRole(ctx, &iam.DeleteRoleInput{
			RoleName: found.RoleName,
		})
		if err != nil {
			return nil, err
		}
	}

	_, subName, ok := strings.Cut(identityProviderARN, "/")
	if !ok {
		return nil, nil
	}
	subName = subName + ":sub"

	assumeRole, err := json.Marshal(policyDocument{
		Version: "2012-10-17",
		Statement: []statement{
			{
				Effect: "Allow",
				Action: []string{"sts:AssumeRoleWithWebIdentity"},
				Principal: map[string]any{
					"Federated": identityProviderARN,
				},
				Condition: map[string]any{
					"StringLike": map[string]any{
						subName: fmt.Sprintf("system:serviceaccount:%s:%s", app.Status.Namespace, serviceName),
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	role, err := client.CreateRole(ctx, &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(assumeRole)),
		RoleName:                 &roleName,
		Description:              aws.String(fmt.Sprintf("Generated role for acorn app %s/%s service %s", app.Namespace, publicname.Get(app), serviceName)),
		MaxSessionDuration:       aws.Int32(3600),
		Tags:                     getTags(app, serviceName),
	})
	if err != nil {
		return nil, err
	}

	return role.Role, nil
}

func getTags(app *v1.AppInstance, serviceName string) []types.Tag {
	return []types.Tag{
		{
			Key:   aws.String(labels.AcornManaged),
			Value: aws.String("true"),
		},
		{
			Key:   aws.String(labels.AcornAppNamespace),
			Value: aws.String(app.Namespace),
		},
		{
			Key:   aws.String(labels.AcornAppName),
			Value: aws.String(app.Name),
		},
		{
			Key:   aws.String(labels.AcornPublicName),
			Value: aws.String(publicname.Get(app)),
		},
		{
			Key:   aws.String(labels.AcornServiceName),
			Value: aws.String(serviceName),
		},
	}
}

func createPolicy(ctx context.Context, client *iam.Client, app *v1.AppInstance, serviceName, roleName, policy string) (string, error) {
	policyName := name.SafeHashConcatName("acorn", app.Namespace, app.Name, serviceName)

	var (
		found  *types.Policy
		marker *string
	)

outer:
	for {
		policies, err := client.ListPolicies(ctx, &iam.ListPoliciesInput{
			Scope:  types.PolicyScopeTypeLocal,
			Marker: marker,
		})
		if err != nil {
			return "", err
		}

		for _, policy := range policies.Policies {
			if policy.PolicyName != nil && *policy.PolicyName == policyName {
				found = &policy
				break outer
			}
		}

		if policies.IsTruncated && policies.Marker != nil {
			marker = policies.Marker
			continue
		}
		break
	}

	if found != nil {
		// ignored error
		_, _ = client.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
			PolicyArn: found.Arn,
			RoleName:  &roleName,
		})
		_, err := client.DeletePolicy(ctx, &iam.DeletePolicyInput{
			PolicyArn: found.Arn,
		})
		if err != nil {
			return "", err
		}
	}

	output, err := client.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyDocument: &policy,
		PolicyName:     &policyName,
		Description:    aws.String(fmt.Sprintf("Generated policy for acorn app %s/%s service %s", app.Namespace, publicname.Get(app), serviceName)),
		Tags:           getTags(app, serviceName),
	})
	if err != nil {
		return "", err
	}

	return *output.Policy.Arn, nil
}

func getAssumeRole(permissions v1.Permissions) string {
	for _, rule := range permissions.Rules {
		if slices.Contains(rule.APIGroups, AWSRoleAPIGroup) &&
			slices.Contains(rule.Scopes, AssumeVerb) &&
			len(rule.Resources) == 0 {
			return rule.Resources[0]
		}
	}
	return ""
}

func alreadyCreated(ctx context.Context, c kclient.Client, app *v1.AppInstance, serviceName, roleHash string) (string, error) {
	sa := &corev1.ServiceAccount{}
	err := c.Get(ctx, router.Key(app.Status.Namespace, serviceName), sa)
	if err == nil {
		if sa.Annotations[AWSRoleHash] == roleHash {
			return sa.Annotations[AssumeRoleAnnotation], nil
		}
	} else if !apierror.IsNotFound(err) {
		return "", err
	}
	return "", nil
}

func AWSAnnotations(ctx context.Context, c kclient.Client, app *v1.AppInstance, permissions v1.Permissions, serviceName string) (map[string]string, error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return nil, err
	}

	if app.Spec.GetStopped() {
		// strip privileges for stopped app
		return nil, nil
	}

	if cfg.AWSIdentityProviderARN == nil || *cfg.AWSIdentityProviderARN == "" {
		return nil, err
	}

	assumeRole := getAssumeRole(permissions)
	if assumeRole != "" {
		return map[string]string{
			AssumeRoleAnnotation: assumeRole,
		}, nil
	}

	policy, err := toPolicy(permissions)
	if err != nil {
		return nil, err
	}

	// no policy, skip
	if policy == "" {
		return nil, err
	}

	awscfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := iam.NewFromConfig(awscfg)

	roleARN, err := getRoleARN(ctx, c, client, app, *cfg.AWSIdentityProviderARN, serviceName, policy)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		AWSRoleHash:          digest.SHA256(policy),
		AssumeRoleAnnotation: roleARN,
	}, nil
}
