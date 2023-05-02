package publicname

import "strings"

func SplitPodContainerName(name string) (string, string) {
	parts := strings.Split(name, ".")
	last := parts[len(parts)-1]
	podName, containerName, _ := strings.Cut(last, ":")
	return podName, containerName
}
