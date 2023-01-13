package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	acornv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/cli/testdata"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestAppsThenContainersCompletion(t *testing.T) {
	appNames := []string{"test-1", "acorn-1", "acorn-2", "hub", "test-2"}
	containerNames := []string{"test-1.container-1", "acorn-1.container-1", "acorn-2.container-1", "hub.container", "hub.other-container", "test-2.container-1"}
	apps := make([]apiv1.App, 0, len(appNames))
	for _, name := range appNames {
		apps = append(apps, apiv1.App{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}
	containers := make([]apiv1.ContainerReplica, 0, len(containerNames))
	for _, name := range containerNames {
		containers = append(containers, apiv1.ContainerReplica{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: apiv1.ContainerReplicaSpec{AppName: strings.Split(name, ".")[0]}})
	}
	mockClientFactory := &testdata.MockClientFactory{
		AppList:       apps,
		ContainerList: containers,
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		args          []string
		toComplete    string
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "Nothing to complete, return all app names",
			wantNames:     appNames,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with t, only app names returned",
			toComplete:    "t",
			wantNames:     []string{"test-1", "test-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with t, but test-1 already in args",
			toComplete:    "t",
			args:          []string{"test-1"},
			wantNames:     []string{"test-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with a, but acorn-1 and acorn-2 already in args",
			toComplete:    "a",
			args:          []string{"acorn-2", "acorn-1"},
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete hub, only hub returned",
			toComplete:    "hub",
			wantNames:     []string{"hub"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete empty, but all names already in args",
			args:          appNames,
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with hub.",
			toComplete:    "hub.",
			wantNames:     []string{"hub.container", "hub.other-container"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with hub.c",
			toComplete:    "hub.c",
			wantNames:     []string{"hub.container"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete hub.container, only hub.container returned",
			toComplete:    "hub.container",
			wantNames:     []string{"hub.container"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete hub.c, but hub.container already in args",
			args:          []string{"hub.container"},
			toComplete:    "hub.c",
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete empty, but all container names already in args",
			args:          containerNames,
			wantNames:     appNames,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete something that doesn't exist",
			toComplete:    "hello",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	comp := newCompletion(mockClientFactory, appsThenContainersCompletion)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := comp.complete(cmd, tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "appsThenContainersCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantDirective, got1, "appsThenContainersCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
		})
	}
}

func TestAppsCompletion(t *testing.T) {
	names := []string{"test-1", "acorn-1", "acorn-2", "hub", "test-2"}
	apps := make([]apiv1.App, 0, len(names))
	for _, name := range names {
		apps = append(apps, apiv1.App{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}
	mockClientFactory := &testdata.MockClientFactory{
		AppList: apps,
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		args          []string
		toComplete    string
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "Nothing to complete, return all",
			wantNames:     names,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with t",
			toComplete:    "t",
			wantNames:     []string{"test-1", "test-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with t, but test-1 already in args",
			toComplete:    "t",
			args:          []string{"test-1"},
			wantNames:     []string{"test-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with a, but acorn-1 and acorn-2 already in args",
			toComplete:    "a",
			args:          []string{"acorn-2", "acorn-1"},
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete hub, only hub returned",
			toComplete:    "hub",
			wantNames:     []string{"hub"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete empty, but all names already in args",
			args:          names,
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete something that doesn't exist",
			toComplete:    "hello",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	comp := newCompletion(mockClientFactory, appsCompletion)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := comp.complete(cmd, tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "appsCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantDirective, got1, "appsCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
		})
	}
}

func TestContainersCompletion(t *testing.T) {
	names := []string{"test-1.container-1", "acorn-1.container-1", "acorn-2.container-1", "hub.container", "hub.other-container", "test-2.container-1"}
	containers := make([]apiv1.ContainerReplica, 0, len(names))
	for _, name := range names {
		containers = append(containers, apiv1.ContainerReplica{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: apiv1.ContainerReplicaSpec{AppName: strings.Split(name, ".")[0]}})
	}
	mockClientFactory := &testdata.MockClientFactory{
		ContainerList: containers,
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		args          []string
		toComplete    string
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "Nothing to complete, return all",
			wantNames:     names,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with t",
			toComplete:    "t",
			wantNames:     []string{"test-1.container-1", "test-2.container-1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with test-1",
			toComplete:    "test-1",
			wantNames:     []string{"test-1.container-1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with t, but test-1.container-1 already in args",
			toComplete:    "t",
			args:          []string{"test-1.container-1"},
			wantNames:     []string{"test-2.container-1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with hub.",
			toComplete:    "hub.",
			wantNames:     []string{"hub.container", "hub.other-container"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with hub.c",
			toComplete:    "hub.c",
			wantNames:     []string{"hub.container"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete hub.container, only hub.container returned",
			toComplete:    "hub.container",
			wantNames:     []string{"hub.container"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete empty, but all names already in args",
			args:          names,
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete app that doesn't exist",
			toComplete:    "hello",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete container that doesn't exist",
			toComplete:    "hello.bye",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	comp := newCompletion(mockClientFactory, containersCompletion)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := comp.complete(cmd, tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "containersCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantDirective, got1, "containersCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
		})
	}
}

func TestWithSuccessDirective(t *testing.T) {
	names := []string{"success"}
	mockClientFactory := &testdata.MockClientFactory{}
	cf := func(_ context.Context, _ client.Client, toComplete string) ([]string, error) {
		if toComplete == "error" {
			return nil, fmt.Errorf("error")
		}
		return names, nil
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name             string
		toComplete       string
		successDirective cobra.ShellCompDirective
		wantNames        []string
		wantDirective    cobra.ShellCompDirective
	}{
		{
			name:             "Default success directive",
			successDirective: cobra.ShellCompDirectiveDefault,
			wantNames:        names,
			wantDirective:    cobra.ShellCompDirectiveDefault,
		},
		{
			name:             "NoSpace success directive",
			successDirective: cobra.ShellCompDirectiveNoSpace,
			wantNames:        names,
			wantDirective:    cobra.ShellCompDirectiveNoSpace,
		},
		{
			name:             "NoSpace success directive, but error occurs",
			successDirective: cobra.ShellCompDirectiveNoSpace,
			toComplete:       "error",
			wantDirective:    cobra.ShellCompDirectiveError,
		},
	}

	comp := newCompletion(mockClientFactory, cf)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := comp.withSuccessDirective(tt.successDirective).complete(cmd, nil, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "onlyNumArgs returned names %v, wanted %v", got, tt.wantNames)
			assert.Equalf(t, tt.wantDirective, got1, "onlyNumArgs returned directive %v, wanted %v", got1, tt.wantDirective)
		})
	}
}

func TestOnlyNumArgsCompletion(t *testing.T) {
	names := []string{"fail"}
	mockClientFactory := &testdata.MockClientFactory{}
	cf := func(context.Context, client.Client, string) ([]string, error) {
		return names, nil
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		args          []string
		numArgs       int
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "max one arg with no args",
			numArgs:       1,
			args:          []string{},
			wantNames:     names,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "max one arg with one arg",
			numArgs:       1,
			args:          []string{""},
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveDefault,
		},
		{
			name:          "max ten args with one arg",
			numArgs:       10,
			args:          []string{""},
			wantNames:     names,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:    "max ten args with nine args",
			numArgs: 10,
			// Will have length 9
			args:          strings.Split(strings.Repeat(".", 8), "."),
			wantNames:     names,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:    "max ten args with ten args",
			numArgs: 10,
			// Will have length 10
			args:          strings.Split(strings.Repeat(".", 9), "."),
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveDefault,
		},
		{
			name:    "max ten args with twenty args",
			numArgs: 10,
			// Will have length 20
			args:          strings.Split(strings.Repeat(".", 19), "."),
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := newCompletion(mockClientFactory, cf).withShouldCompleteOptions(onlyNumArgs(tt.numArgs)).complete(cmd, tt.args, "")
			assert.Equalf(t, tt.wantNames, got, "onlyNumArgs returned names %v, wanted %v", got, tt.wantNames)
			assert.Equalf(t, tt.wantDirective, got1, "onlyNumArgs returned directive %v, wanted %v", got1, tt.wantDirective)
		})
	}
}

func TestAcornContainerCompletion(t *testing.T) {
	names := map[string]map[string]acornv1.Container{
		"test-1": {
			"container-1": {},
			"container-2": {},
		},
		"test-2": {
			"container-1": {},
		},
		"hub": {
			"web":        {},
			"db":         {},
			"controller": {},
		},
	}
	apps := make([]apiv1.App, 0, len(names))
	for _, entry := range typed.Sorted(names) {
		apps = append(apps, apiv1.App{
			ObjectMeta: metav1.ObjectMeta{Name: entry.Key},
			Status: acornv1.AppInstanceStatus{
				AppSpec: acornv1.AppSpec{Containers: entry.Value},
			},
		})
	}
	mockClientFactory := &testdata.MockClientFactory{
		AppList: apps,
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		toComplete    string
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "Nothing to complete, return all",
			wantNames:     []string{"controller", "db", "web", "container-1", "container-2", "container-1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Completing c",
			toComplete:    "c",
			wantNames:     []string{"controller", "container-1", "container-2", "container-1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Completing container",
			toComplete:    "container",
			wantNames:     []string{"container-1", "container-2", "container-1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	comp := newCompletion(mockClientFactory, acornContainerCompletion)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := comp.complete(cmd, nil, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "acornContainerCompletion(_, _, nil, %v)", tt.toComplete)
			assert.Equalf(t, tt.wantDirective, got1, "acornContainerCompletion(_, _, nil, %v)", tt.toComplete)
		})
	}
}

func TestOnlyAppsWithAcornContainer(t *testing.T) {
	names := map[string]map[string]acornv1.Container{
		"test-1": {
			"container-1": {},
			"container-2": {},
		},
		"test-2": {
			"container-1": {},
		},
		"hub": {
			"web":        {},
			"db":         {},
			"controller": {},
		},
	}
	apps := make([]apiv1.App, 0, len(names))
	containers := make([]apiv1.ContainerReplica, 0, len(names))
	for _, entry := range typed.Sorted(names) {
		appName := entry.Key
		apps = append(apps, apiv1.App{
			ObjectMeta: metav1.ObjectMeta{Name: appName},
			Status: acornv1.AppInstanceStatus{
				AppSpec: acornv1.AppSpec{Containers: entry.Value},
			},
		})
		for _, c := range typed.Sorted(entry.Value) {
			containers = append(containers, apiv1.ContainerReplica{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.%s", appName, c.Key)},
				Spec:       apiv1.ContainerReplicaSpec{AppName: appName},
			})
		}
	}
	mockClientFactory := &testdata.MockClientFactory{
		AppList:       apps,
		ContainerList: containers,
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		container     string
		toComplete    string
		args          []string
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "Nothing to complete and no container, return apps",
			wantNames:     []string{"hub", "test-1", "test-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Nothing to complete and no container, return apps except those in args",
			args:          []string{"hub", "test-2"},
			wantNames:     []string{"test-1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Something to complete and no container should be the apps completion",
			toComplete:    "t",
			wantNames:     []string{"test-1", "test-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Something with '.' to complete and no container should be the k8s container completion",
			toComplete:    "hub.",
			wantNames:     []string{"hub.controller", "hub.db", "hub.web"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Something with '.' to complete and no container should be the k8s container completion except those in args",
			toComplete:    "hub.",
			args:          []string{"hub.db"},
			wantNames:     []string{"hub.controller", "hub.web"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Nothing to complete and 'container-1' for container should produce 'test' apps",
			container:     "container-1",
			wantNames:     []string{"test-1", "test-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Nothing to complete and 'container-1' for container should produce 'test' apps except those in args",
			container:     "container-1",
			args:          []string{"test-1"},
			wantNames:     []string{"test-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "'test-1' to complete and 'container-1' for container should produce test-1",
			container:     "container-1",
			toComplete:    "test-1",
			wantNames:     []string{"test-1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Container that doesn't exist should produce nil completions",
			container:     "container",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Container exists, but no app completion should produce nil completions",
			container:     "db",
			toComplete:    "t",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := newCompletion(mockClientFactory, onlyAppsWithAcornContainer(tt.container)).complete(cmd, tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "onlyAppsWithAcornContainer(_, _, nil, %v)", tt.toComplete)
			assert.Equalf(t, tt.wantDirective, got1, "onlyAppsWithAcornContainer(_, _, nil, %v)", tt.toComplete)
		})
	}
}

func TestImagesCompletion(t *testing.T) {
	mockClientFactory := &testdata.MockClientFactory{}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		allowDigest   bool
		toComplete    string
		args          []string
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "Nothing to complete, return digest or tag but not both",
			allowDigest:   true,
			wantNames:     []string{"testtag:latest", "lkjhgfdsa098", "testtag1:latest", "testtag2:v1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Nothing to complete, return only tags",
			allowDigest:   false,
			wantNames:     []string{"testtag:latest", "testtag1:latest", "testtag2:v1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Match image digest even if tag exists",
			allowDigest:   true,
			toComplete:    "l",
			wantNames:     []string{"lkjhgfdsa098", "lkjhgfdsa123"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Don't allow matching of image digest if tag doesn't exist",
			allowDigest:   false,
			toComplete:    "l",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Return all tags matched, even for same image",
			allowDigest:   true,
			toComplete:    "testtag",
			wantNames:     []string{"testtag:latest", "testtag1:latest", "testtag2:v1"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Don't return matched tag that is already in args",
			allowDigest:   true,
			args:          []string{"testtag2:v1"},
			toComplete:    "testtag",
			wantNames:     []string{"testtag:latest", "testtag1:latest"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Don't return matched digest that is already in args",
			allowDigest:   true,
			args:          []string{"lkjhgfdsa098"},
			toComplete:    "l",
			wantNames:     []string{"lkjhgfdsa123"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "No matches",
			allowDigest:   true,
			toComplete:    "acorn",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := newCompletion(mockClientFactory, imagesCompletion(tt.allowDigest)).complete(cmd, tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "imagesCompletion(%v)(_, _, nil, %v)", tt.allowDigest, tt.toComplete)
			assert.Equalf(t, tt.wantDirective, got1, "imagesCompletion(%v)(_, _, nil, %v)", tt.allowDigest, tt.toComplete)
		})
	}
}

func TestCredentialsCompletion(t *testing.T) {
	names := []string{"docker.io", "ghcr.io", "great.io"}
	creds := make([]apiv1.Credential, 0, len(names))
	for _, name := range names {
		creds = append(creds, apiv1.Credential{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}
	mockClientFactory := &testdata.MockClientFactory{
		CredentialList: creds,
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		args          []string
		toComplete    string
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "Nothing to complete, return all",
			wantNames:     names,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with g",
			toComplete:    "g",
			wantNames:     []string{"ghcr.io", "great.io"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with g, but great.io already in args",
			toComplete:    "g",
			args:          []string{"great.io"},
			wantNames:     []string{"ghcr.io"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with g, but all g's in args",
			toComplete:    "g",
			args:          []string{"ghcr.io", "great.io"},
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete docker.io, only docker.io returned",
			toComplete:    "docker.io",
			wantNames:     []string{"docker.io"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete empty, but all names already in args",
			args:          names,
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete something that doesn't exist",
			toComplete:    "hello",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	comp := newCompletion(mockClientFactory, credentialsCompletion)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := comp.complete(cmd, tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "credentialsCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantDirective, got1, "credentialsCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
		})
	}
}

func TestVolumesCompletion(t *testing.T) {
	names := []string{"acorn.volume-1", "acorn.volume-2", "my-volume", "empty"}
	volumes := make([]apiv1.Volume, 0, len(names))
	for _, name := range names {
		volumes = append(volumes, apiv1.Volume{ObjectMeta: metav1.ObjectMeta{Name: name}})
	}
	mockClientFactory := &testdata.MockClientFactory{
		VolumeList: volumes,
	}
	cmd := new(cobra.Command)
	cmd.SetContext(context.Background())

	tests := []struct {
		name          string
		args          []string
		toComplete    string
		wantNames     []string
		wantDirective cobra.ShellCompDirective
	}{
		{
			name:          "Nothing to complete, return all",
			wantNames:     names,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with a",
			toComplete:    "a",
			wantNames:     []string{"acorn.volume-1", "acorn.volume-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with a, but acorn.volume-1 already in args",
			toComplete:    "a",
			args:          []string{"acorn.volume-1"},
			wantNames:     []string{"acorn.volume-2"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete starting with a, but all a's in args",
			toComplete:    "a",
			args:          []string{"acorn.volume-2", "acorn.volume-1"},
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete my-volume, only my-volume returned",
			toComplete:    "my-volume",
			wantNames:     []string{"my-volume"},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete empty, but all names already in args",
			args:          names,
			wantNames:     []string{},
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:          "Complete something that doesn't exist",
			toComplete:    "hello",
			wantNames:     nil,
			wantDirective: cobra.ShellCompDirectiveNoFileComp,
		},
	}

	comp := newCompletion(mockClientFactory, volumesCompletion)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := comp.complete(cmd, tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantNames, got, "volumesCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
			assert.Equalf(t, tt.wantDirective, got1, "volumesCompletion(_, _, %v, %v)", tt.args, tt.toComplete)
		})
	}
}
