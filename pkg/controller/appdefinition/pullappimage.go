package appdefinition

import (
	"encoding/json"

	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/ibuildthecloud/baaah/pkg/typed"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/condition"
	herdlabels "github.com/ibuildthecloud/herd/pkg/labels"
	"github.com/pkg/errors"
	"github.com/rancher/wrangler/pkg/name"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PullAppImage(appImageInitImage string) router.HandlerFunc {
	return func(req router.Request, resp router.Response) error {
		return pullAppImage(appImageInitImage, req, resp)
	}
}

type appImageWithError struct {
	v1.AppImage
	Error string `json:"error,omitempty"`
}

func getOutput(client router.Client, image, name, namespace string) (result appImageWithError, _ error) {
	job, err := typed.Get[*batchv1.Job](client, name, &meta.GetOptions{
		Namespace: namespace,
	})
	if apierror.IsNotFound(err) {
		return result, nil
	} else if err != nil {
		return result, err
	}

	if job.Annotations[herdlabels.HerdAppImage] != image {
		return result, nil
	}

	if job.Status.Succeeded != 1 {
		return result, nil
	}

	sel, err := metav1.LabelSelectorAsSelector(job.Spec.Selector)
	if err != nil {
		return result, err
	}

	pods, err := typed.List[*corev1.PodList](client, &meta.ListOptions{
		Namespace: namespace,
		Selector:  sel,
	})
	if err != nil {
		return result, err
	}

	if len(pods.Items) == 0 {
		return result, nil
	}

	for _, pod := range pods.Items {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name != "image" {
				continue
			}
			if status.State.Terminated == nil || status.State.Terminated.ExitCode != 0 {
				continue
			}
			if err := json.Unmarshal([]byte(status.State.Terminated.Message), &result); err != nil {
				return result, err
			}
			return result, nil
		}
	}

	return result, nil
}

func pullAppImage(appImageInitImage string, req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	if appInstance.Status.Namespace == "" {
		return nil
	}

	cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionPulled)

	if appInstance.Spec.Image == appInstance.Status.AppImage.ID {
		cond.Success()
		return nil
	}

	jobName, jobNS := name.SafeConcatName(appInstance.Name, "pull"), appInstance.Status.Namespace

	appImage, err := getOutput(req.Client, appInstance.Spec.Image, jobName, jobNS)
	if err != nil {
		return err
	}

	if appImage.Error != "" {
		cond.Error(errors.New(appImage.Error))
		return nil
	}

	if appImage.ID == "" {
		cond.Unknown()
		resp.Objects(pullJob(appInstance.Spec.Image, appImageInitImage, jobName, jobNS))
	} else {
		cond.Success()
		appInstance.Status.AppImage = appImage.AppImage
		resp.Objects(appInstance)
	}

	return nil
}

func pullJob(appImage, initImage, name, namespace string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				herdlabels.HerdAppImage: appImage,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &[]int32{3}[0],
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					InitContainers: []corev1.Container{
						{
							Name:  "init",
							Image: initImage,
							Args: []string{
								"init",
								"/share/init",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "share",
									MountPath: "/share",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "image",
							Image: appImage,
							Env: []corev1.EnvVar{
								{
									Name:  "ID",
									Value: appImage,
								},
								{
									Name:  "OUTPUT",
									Value: "/dev/termination-log",
								},
							},
							Command: []string{
								"/share/init",
							},
							Args: []string{},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "share",
									MountPath: "/share",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "share",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}
