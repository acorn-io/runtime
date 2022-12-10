package system

import "os"

func IsRunningAsPod() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}
