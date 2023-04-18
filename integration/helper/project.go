package helper

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/stretchr/testify/assert"
	"gotest.tools/v3/icmd"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	sharedlock sync.Mutex
)

// TempProject creates a new Kubernetes Namespace object with a generated name starting with "acorn-project-"
// and sets some labels on it. This function then calls the tempCreateNamespaceHelper function to create the Namespace
// using the provided Kubernetes client. The created Namespace object is returned.
func TempProject(t *testing.T, client client.Client) *corev1.Namespace {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			// namespace ends up as "acorn-test-{random chars}"
			GenerateName: "acorn-project-",
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
	}
	return tempCreateNamespaceHelper(t, client, ns)
}

func NamedTempProject(t *testing.T, client client.Client, name string) *corev1.Namespace {
	t.Helper()
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"test.acorn.io/namespace": "true",
				labels.AcornProject:       "true",
			},
		},
	}
	return tempCreateNamespaceHelper(t, client, ns)
}

// SwitchCLIDefaultProjectWithSwitchbackAtCleanup switches the default project for the acorn CLI
// to a new project with the given name, and sets up a cleanup function to switch back to the original default
// project at the end of the test. If there was no original default project, the cleanup function will not be set up.
// This has the possibility to race from multiple packages calling this function and running in parallel
func SwitchCLIDefaultProjectWithSwitchbackAtCleanup(t *testing.T, newProjectName string) {
	t.Helper()
	sharedlock.Lock()
	// Find the current Project
	result := icmd.RunCommand("acorn", "project", "-o=json")
	projectJSONs := strings.Split(result.Stdout(), "\n\n")
	var defaultProject string
	for _, p := range projectJSONs {
		existingProject := TestProject{}
		// don't parse empty responses
		if len(p) == 0 {
			continue
		}
		if err := json.Unmarshal([]byte(p), &existingProject); err != nil {
			fmt.Printf("Could not unmarshal project from cli: %s\n", p)
			continue
		}
		if existingProject.Default {
			defaultProject = existingProject.Name
			break
		}
	}
	// switch current default project to user specified newProjectName
	SwitchCLIDefaultProject(t, newProjectName)
	// cleanup function to switch back to original default project at end of test
	// don't switch back if there was no original default project
	if defaultProject != "" {
		t.Cleanup(func() {
			SwitchCLIDefaultProject(t, defaultProject)
			sharedlock.Unlock()
		})
	} else {
		sharedlock.Unlock()
	}
}

// SwitchCLIDefaultProject switches the default project for the acorn CLI
func SwitchCLIDefaultProject(t *testing.T, projectName string) {
	t.Helper()
	result := icmd.RunCommand("acorn", "project", "use", projectName)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, result.Stderr(), "")
}
