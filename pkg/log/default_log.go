package log

import (
	"context"
	"sync"
	"time"

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
	lastLoginTime       time.Time
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

		// Prompt the user to fill in a credential secret if:
		// - The app status indicates a login is needed, AND
		// - The app has been updated since the last time the user logged in, AND
		// - (The app is running locally OR the last login was more than 5 seconds ago)
		// The last condition is needed to avoid a race condition in the SaaS where the user is prompted multiple
		// times in a row to fill in a credential secret. We don't want to prompt a user more than once within
		// five seconds in order to avoid the double prompt.
		if app.Status.AppStatus.LoginRequired &&
			app.Status.ObservedGeneration > d.lastLoginGeneration &&
			(app.Status.ResolvedOfferings.Region == apiv1.LocalRegion ||
				time.Since(d.lastLoginTime) > 5*time.Second) {

			if err := login.Secrets(d.ctx, d.client, app); err != nil {
				go d.Errorf(err.Error())
			} else {
				d.lastLoginGeneration = app.Generation
				d.lastLoginTime = time.Now()
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
