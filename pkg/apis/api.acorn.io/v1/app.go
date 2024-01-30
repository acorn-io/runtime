package v1

import (
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   v1.AppInstanceSpec `json:"spec,omitempty"`
	Status AppStatus          `json:"status,omitempty"`
}

func (in *App) GetStopped() bool {
	return in.Spec.Stop != nil && *in.Spec.Stop && in.DeletionTimestamp.IsZero()
}

func (in *App) GetRegion() string {
	if in.Spec.Region != "" {
		return in.Spec.Region
	}
	return in.Status.Defaults.Region
}

type AppStatus v1.EmbeddedAppStatus

func (in *AppStatus) Condition(name string) v1.Condition {
	for _, cond := range in.Conditions {
		if cond.Type == name {
			return cond
		}
	}
	return v1.Condition{}
}

func (in *AppStatus) GetDevMode() bool {
	return in.DevSession != nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

// AppToAppInstance converts an App to a [v1.AppInstance] and returns the result.
func AppToAppInstance(in *App) *v1.AppInstance {
	return &v1.AppInstance{
		ObjectMeta: in.ObjectMeta,
		Spec:       in.Spec,
		Status: v1.AppInstanceStatus{
			EmbeddedAppStatus: v1.EmbeddedAppStatus(in.Status),
		},
	}
}

// AppInstanceToApp converts a [v1.AppInstance] to an App and returns the result.
func AppInstanceToApp(in *v1.AppInstance) *App {
	return &App{
		ObjectMeta: in.ObjectMeta,
		Spec:       in.Spec,
		Status:     AppStatus(in.Status.EmbeddedAppStatus),
	}
}
