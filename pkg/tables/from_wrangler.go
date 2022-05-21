package tables

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/go-units"
	"github.com/rancher/wrangler-cli/pkg/table/types"
	"github.com/rancher/wrangler/pkg/data/convert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

var (
	localFuncMap = map[string]interface{}{
		"ago":         FormatCreated,
		"json":        FormatJSON,
		"jsoncompact": FormatJSONCompact,
		"yaml":        FormatYAML,
		"first":       FormatFirst,
		"dump":        FormatSpew,
		"toJson":      ToJSON,
		"boolToStar":  BoolToStar,
		"array":       ToArray,
		"arrayFirst":  ToArrayFirst,
		"graph":       Graph,
		"pointer":     Pointer,
	}
)

func ToArray(s []string) (string, error) {
	return strings.Join(s, ", "), nil
}

func ToArrayFirst(s []string) (string, error) {
	if len(s) > 0 {
		return s[0], nil
	}
	return "", nil
}

func Graph(value int) (string, error) {
	bars := int(float64(value) / 100.0 * 30)
	builder := &strings.Builder{}
	for i := 0; i < bars; i++ {
		if i == bars-1 {
			builder.WriteString(fmt.Sprintf("> %v", value))
			break
		}
		builder.WriteString("=")
	}
	return builder.String(), nil
}

func Pointer(data interface{}) string {
	if reflect.ValueOf(data).IsNil() {
		return ""
	}
	return fmt.Sprint(data)
}

func FormatID(obj runtime.Object, namespace string) (string, error) {
	r, err := types.FromObject(obj)
	return r.StringDefaultNamespace(namespace), err
}

func FormatCreated(data interface{}) (string, error) {
	t, ok := data.(metav1.Time)
	if !ok {
		return "", nil
	}

	return units.HumanDuration(time.Now().UTC().Sub(t.Time)) + " ago", nil
}

func FormatJSON(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "    ")
	return string(bytes) + "\n", err
}

func FormatJSONCompact(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	return string(bytes) + "\n", err
}

func FormatYAML(data interface{}) (string, error) {
	bytes, err := yaml.Marshal(data)
	return string(bytes) + "\n", err
}

func FormatSpew(data interface{}) (string, error) {
	return spew.Sdump(data), nil
}

func FormatFirst(data, data2 interface{}) (string, error) {
	str := convert.ToString(data)
	if str != "" {
		return str, nil
	}

	str = convert.ToString(data2)
	if str != "" {
		return str, nil
	}

	return "", nil
}

func ToJSON(data interface{}) (map[string]interface{}, error) {
	return convert.EncodeToMap(data)
}

func BoolToStar(obj interface{}) (string, error) {
	if b, ok := obj.(bool); ok && b {
		return "*", nil
	}
	if b, ok := obj.(*bool); ok && b != nil && *b {
		return "*", nil
	}
	return "", nil
}
