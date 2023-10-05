package log

import (
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var DefaultLogger = NewDefaultLogger()

func NewDefaultLogger() *DefaultLoggerImpl {
	return &DefaultLoggerImpl{
		containerColors: map[string]pterm.Color{},
	}
}

type DefaultLoggerImpl struct {
	containerColors map[string]pterm.Color
}

func (*DefaultLoggerImpl) Errorf(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

func (*DefaultLoggerImpl) Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

func (*DefaultLoggerImpl) AppStatus(ready bool, msg string) {
	if ready {
		pterm.DefaultBox.Println(pterm.LightGreen(msg))
	} else {
		pterm.Println(pterm.LightYellow(msg))
	}
}

func (d *DefaultLoggerImpl) Container(timeStamp metav1.Time, containerName, line string) {
	color, ok := d.containerColors[containerName]
	if !ok {
		color = nextColor()
		d.containerColors[containerName] = color
	}
	pterm.Printf("%s: %s\n", color.Sprint(containerName), line)
}
