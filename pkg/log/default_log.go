package log

import (
	"context"
	"sync"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/login"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDefaultLogger(ctx context.Context, c client.Client) *DefaultLoggerImpl {
	return &DefaultLoggerImpl{
		client:          c,
		ctx:             ctx,
		containerColors: map[string]pterm.Color{},
	}
}

type DefaultLoggerImpl struct {
	lock            sync.Mutex
	ctx             context.Context
	client          client.Client
	containerColors map[string]pterm.Color
	lastLogin       int64
}

func (d *DefaultLoggerImpl) Errorf(format string, args ...interface{}) {
	d.lock.Lock()
	defer d.lock.Unlock()
	logrus.Errorf(format, args...)
}

func (d *DefaultLoggerImpl) Infof(format string, args ...interface{}) {
	d.lock.Lock()
	defer d.lock.Unlock()
	logrus.Infof(format, args...)
}

func (d *DefaultLoggerImpl) AppStatus(ready bool, msg string, app *apiv1.App) {
	if ready {
		pterm.DefaultBox.Println(pterm.LightGreen(msg))
	} else {
		d.lock.Lock()
		defer d.lock.Unlock()
		if app.Status.AppStatus.LoginRequired && app.Status.ObservedGeneration > d.lastLogin {
			// Wait until all containers in the app are defined in order to avoid a race condition.
			// When deploying acorns to the SaaS, a race condition can occur where the user is prompted
			// to enter the credentials for a secret multiple times, as the app changes upstream while the
			// user is typing into the prompt. If we wait until all containers are defined before prompting
			// the user, then the race condition is avoided.
			allDefined := true
			for _, c := range app.Status.AppStatus.Containers {
				if !c.CommonStatus.Defined {
					allDefined = false
					break
				}
			}

			if allDefined {
				err := login.Secrets(d.ctx, d.client, app)
				if err != nil {
					go d.Errorf(err.Error())
				} else {
					d.lastLogin = app.Generation
				}
			}
		}
		pterm.Println(pterm.LightYellow(msg))
	}
}

func (d *DefaultLoggerImpl) Container(timeStamp metav1.Time, containerName, line string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	color, ok := d.containerColors[containerName]
	if !ok {
		color = nextColor()
		d.containerColors[containerName] = color
	}
	pterm.Printf("%s: %s\n", color.Sprint(containerName), line)
}
