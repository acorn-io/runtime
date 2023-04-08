package secrets

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/jobs"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/ref"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func seedData(existing *corev1.Secret, from map[string]string, keys ...string) map[string][]byte {
	to := map[string][]byte{}
	if existing != nil {
		for _, key := range keys {
			to[key] = existing.Data[key]
		}
	}
	for _, key := range keys {
		if v, ok := from[key]; ok {
			// don't override a non-zero length value with zero length
			if len(v) > 0 || len(to[key]) == 0 {
				to[key] = []byte(v)
			}
		}
	}
	return to
}

var (
	ErrJobNotDone        = errors.New("job not complete")
	ErrJobNoOutput       = errors.New("job has no output")
	templateSecretRegexp = regexp.MustCompile(`\${secret://(.*?)/(.*?)}`)
	imageSecretRegexp    = regexp.MustCompile(`\${image://(.*?)}`)
)

func generatedSecret(req router.Request, appInstance *v1.AppInstance, secretName string, secretRef v1.Secret, existing *corev1.Secret) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: secretName + "-",
			Namespace:    appInstance.Namespace,
			Labels:       labelsForSecret(secretName, appInstance, secretRef),
			Annotations:  annotationsForSecret(secretName, appInstance, secretRef),
		},
		Data: seedData(existing, secretRef.Data),
		Type: v1.SecretTypeGenerated,
	}

	format := convert.ToString(secretRef.Params["format"])
	switch format {
	case "text":
		var output string
		_, err := jobs.GetOutputFor(req.Ctx, req.Client, appInstance, convert.ToString(secretRef.Params["job"]), secretName, &output)
		if err != nil {
			return nil, err
		}

		secret.Data["content"] = []byte(output)
	case "aml":
		fallthrough
	case "json":
		newSecret := &v1.Secret{}
		_, err := jobs.GetOutputFor(req.Ctx, req.Client, appInstance, convert.ToString(secretRef.Params["job"]), secretName, newSecret)
		if err != nil {
			return nil, err
		}
		for k, v := range newSecret.Data {
			secret.Data[k] = []byte(v)
		}
		if newSecret.Type != "" {
			inType := corev1.SecretType(v1.SecretTypePrefix + newSecret.Type)
			if v1.SecretTypes[inType] {
				secret.Type = inType
			}
		}
	}

	return updateOrCreate(req, existing, secret)
}

func generateTemplate(secrets map[string]*corev1.Secret, req router.Request, appInstance *v1.AppInstance, secretName string, secretRef v1.Secret, existing *corev1.Secret) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: secretName + "-",
			Namespace:    appInstance.Namespace,
			Labels:       labelsForSecret(secretName, appInstance, secretRef),
			Annotations:  annotationsForSecret(secretName, appInstance, secretRef),
		},
		Data: seedData(existing, secretRef.Data, typed.SortedKeys(secretRef.Data)...),
		Type: v1.SecretTypeTemplate,
	}

	tag, err := images.GetRuntimePullableImageReference(req.Ctx, req.Client, appInstance.Namespace, appInstance.Status.AppImage.ID)
	if err != nil {
		return nil, err
	}

	for _, entry := range typed.Sorted(secret.Data) {
		var (
			template       = string(entry.Value)
			templateErrors []error
		)
		template = templateSecretRegexp.ReplaceAllStringFunc(template, func(t string) string {
			groups := templateSecretRegexp.FindStringSubmatch(t)
			secret, err := GetOrCreateSecret(secrets, req, appInstance, groups[1])
			if err != nil {
				templateErrors = append(templateErrors, err)
				return err.Error()
			}

			val := secret.Data[groups[2]]
			if len(val) == 0 {
				err := fmt.Errorf("failed to find key %s in secret %s", groups[2], groups[1])
				templateErrors = append(templateErrors, err)
				return err.Error()
			}

			return string(val)
		})
		if err := merr.NewErrors(templateErrors...); err != nil {
			return nil, err
		}

		template = imageSecretRegexp.ReplaceAllStringFunc(template, func(t string) string {
			groups := imageSecretRegexp.FindStringSubmatch(t)
			digest, ok := appInstance.Status.AppImage.ImageData.Images[groups[1]]
			if !ok {
				return t
			}

			return images.ResolveTag(tag, digest.Image)
		})

		secret.Data[entry.Key] = []byte(template)
	}

	return updateOrCreate(req, existing, secret)
}

func generateToken(req router.Request, appInstance *v1.AppInstance, secretName string, secretRef v1.Secret, existing *corev1.Secret) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: secretName + "-",
			Namespace:    appInstance.Namespace,
			Labels:       labelsForSecret(secretName, appInstance, secretRef),
			Annotations:  annotationsForSecret(secretName, appInstance, secretRef),
		},
		Data: seedData(existing, secretRef.Data, "token"),
		Type: v1.SecretTypeToken,
	}

	if len(secret.Data["token"]) == 0 {
		length, err := convert.ToNumber(secretRef.Params["length"])
		if err != nil {
			return nil, err
		}
		characters := convert.ToString(secretRef.Params["characters"])
		v, err := generate(characters, int(length))
		if err != nil {
			return nil, err
		}
		secret.Data["token"] = []byte(v)
	}

	return updateOrCreate(req, existing, secret)
}

func generateOpaque(req router.Request, appInstance *v1.AppInstance, secretName string, secretRef v1.Secret, existing *corev1.Secret) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: secretName + "-",
			Namespace:    appInstance.Namespace,
			Labels:       labelsForSecret(secretName, appInstance, secretRef),
			Annotations:  annotationsForSecret(secretName, appInstance, secretRef),
		},
		Data: seedData(existing, secretRef.Data, maps.Keys(secretRef.Data)...),
		Type: v1.SecretTypeOpaque,
	}

	return updateOrCreate(req, existing, secret)
}

func generateBasic(req router.Request, appInstance *v1.AppInstance, secretName string, secretRef v1.Secret, existing *corev1.Secret) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: secretName + "-",
			Namespace:    appInstance.Namespace,
			Labels:       labelsForSecret(secretName, appInstance, secretRef),
			Annotations:  annotationsForSecret(secretName, appInstance, secretRef),
		},
		Data: seedData(existing, secretRef.Data, corev1.BasicAuthUsernameKey, corev1.BasicAuthPasswordKey),
		Type: v1.SecretTypeBasic,
	}

	for i, key := range []string{corev1.BasicAuthUsernameKey, corev1.BasicAuthPasswordKey} {
		if len(secret.Data[key]) == 0 {
			// TODO: Improve with more characters (special, upper/lowercase, etc)
			v, err := randomtoken.Generate()
			v = v[:(i+1)*8]
			if err != nil {
				return nil, err
			}
			secret.Data[key] = []byte(v)
		}
	}
	return updateOrCreate(req, existing, secret)
}

func updateOrCreate(req router.Request, existing, secret *corev1.Secret) (result *corev1.Secret, err error) {
	defer func() {
		if err != nil || result == nil {
			return
		}
		// The result secret should be decrypted, but the written secret in the app namespace should be encrypted
		// if the source data was encrypted
		result = result.DeepCopy()
		result.Data, err = nacl.DecryptNamespacedDataMap(req.Ctx, req.Client, result.Data, result.Namespace)
		if err != nil {
			err = fmt.Errorf("decrypting %s/%s: %w", secret.Namespace, secret.Name, err)
		}
	}()

	if existing == nil {
		return secret, req.Client.Create(req.Ctx, secret)
	}
	if equality.Semantic.DeepEqual(existing.Data, secret.Data) && maps.Equal(existing.Labels, secret.Labels) &&
		maps.Equal(existing.Annotations, secret.Annotations) {
		return existing, nil
	}

	newSecret := existing.DeepCopy()
	newSecret.Data = secret.Data
	newSecret.Annotations = secret.Annotations
	newSecret.Labels = secret.Labels

	return newSecret, req.Client.Update(req.Ctx, newSecret)
}

func acornLabelsForSecret(secretName string, appInstance *v1.AppInstance) map[string]string {
	return map[string]string{
		labels.AcornAppName:         appInstance.Name,
		labels.AcornManaged:         "true",
		labels.AcornSecretName:      secretName,
		labels.AcornSecretGenerated: "true",
	}
}

func labelsForSecret(secretName string, appInstance *v1.AppInstance, secretRef v1.Secret) map[string]string {
	result := labels.Merge(acornLabelsForSecret(secretName, appInstance),
		labels.GatherScoped(secretName, v1.LabelTypeSecret,
			appInstance.Status.AppSpec.Labels, secretRef.Labels, appInstance.Spec.Labels))
	return labels.Merge(result, map[string]string{
		labels.AcornPublicName: publicname.ForChild(appInstance, secretName),
	})
}

func annotationsForSecret(secretName string, appInstance *v1.AppInstance, secretRef v1.Secret) map[string]string {
	return labels.GatherScoped(secretName, v1.LabelTypeSecret, appInstance.Status.AppSpec.Annotations, secretRef.Annotations,
		appInstance.Spec.Annotations)
}

func getSecret(req router.Request, appInstance *v1.AppInstance, name string) (*corev1.Secret, error) {
	l := acornLabelsForSecret(name, appInstance)

	var secrets corev1.SecretList
	err := req.List(&secrets, &kclient.ListOptions{
		Namespace:     appInstance.Namespace,
		LabelSelector: klabels.SelectorFromSet(l),
	})
	if err != nil {
		return nil, err
	}

	if len(secrets.Items) == 0 {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    "v1",
			Resource: "secrets",
		}, name)
	}

	sort.Slice(secrets.Items, func(i, j int) bool {
		return secrets.Items[i].UID < secrets.Items[j].UID
	})

	return &secrets.Items[0], nil
}

func generateSecret(secrets map[string]*corev1.Secret, req router.Request, appInstance *v1.AppInstance, secretName string) (*corev1.Secret, error) {
	existing, err := getSecret(req, appInstance, secretName)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	secretRef, ok := appInstance.Status.AppSpec.Secrets[secretName]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    "v1",
			Resource: "secrets",
		}, secretName)
	}

	switch secretRef.Type {
	case "opaque":
		return generateOpaque(req, appInstance, secretName, secretRef, existing)
	case "basic":
		return generateBasic(req, appInstance, secretName, secretRef, existing)
	case "generated":
		return generatedSecret(req, appInstance, secretName, secretRef, existing)
	case "token":
		return generateToken(req, appInstance, secretName, secretRef, existing)
	case "template":
		return generateTemplate(secrets, req, appInstance, secretName, secretRef, existing)
	default:
		return nil, err
	}
}

func GetOrCreateSecret(secrets map[string]*corev1.Secret, req router.Request, appInstance *v1.AppInstance, secretName string) (*corev1.Secret, error) {
	if sec, ok := secrets[secretName]; ok {
		return sec, nil
	}

	externalRef := ""
	for _, binding := range appInstance.Spec.Secrets {
		if binding.Target == secretName {
			externalRef = binding.Secret
		}
	}

	if externalRef == "" {
		secretDef := appInstance.Status.AppSpec.Secrets[secretName]
		if secretDef.External != "" {
			externalRef = secretDef.External
		}
	}

	if externalRef != "" {
		if strings.HasPrefix(externalRef, "context://") {
			existingSecret := &corev1.Secret{}
			name := "context-" + strings.TrimPrefix(externalRef, "context://")
			if err := req.Get(existingSecret, system.Namespace, name); err != nil {
				return nil, err
			}
			if existingSecret.Type != apiv1.SecretTypeContext {
				return nil, fmt.Errorf("found secrets %s/%s but type is [%s] and not [%s]",
					system.Namespace, name, existingSecret.Type, apiv1.SecretTypeContext)
			}
			return existingSecret, nil
		}
		existingSecret := &corev1.Secret{}
		err := ref.Lookup(req.Ctx, req.Client, existingSecret, appInstance.Namespace, strings.Split(externalRef, ".")...)
		if err != nil {
			return nil, err
		}
		existingSecret = existingSecret.DeepCopy()
		existingSecret.Data, err = nacl.DecryptNamespacedDataMap(req.Ctx, req.Client, existingSecret.Data, appInstance.Namespace)
		if err != nil {
			return nil, err
		}
		secrets[secretName] = existingSecret
		return existingSecret, nil
	}

	secret, err := generateSecret(secrets, req, appInstance, secretName)
	if err != nil {
		return nil, err
	}
	secrets[secretName] = secret
	return secret, nil
}

func generate(characters string, tokenLength int) (string, error) {
	token := make([]byte, tokenLength)
	for i := range token {
		r, err := rand.Int(rand.Reader, big.NewInt(int64(len(characters))))
		if err != nil {
			return "", err
		}
		token[i] = characters[r.Int64()]
	}
	return string(token), nil
}
