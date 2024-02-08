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
	lock                sync.Mutex
	ctx                 context.Context
	client              client.Client
	containerColors     map[string]pterm.Color
	lastLoginGeneration int64
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
		if app.Status.AppStatus.LoginRequired && app.Status.ObservedGeneration > d.lastLoginGeneration {
			updatedApp, err := login.Secrets(d.ctx, d.client, app)
			if err != nil {
				go d.Errorf(err.Error())
			} else {
				d.lastLoginGeneration = updatedApp.Generation
			}
		}
		pterm.Println(pterm.LightYellow(msg))
	}
}

func (d *DefaultLoggerImpl) Container(_ metav1.Time, containerName, line string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	color, ok := d.containerColors[containerName]
	if !ok {
		color = nextColor()
		d.containerColors[containerName] = color
	}
	pterm.Printf("%s: %s\n", color.Sprint(containerName), line)
}
