package cosign

const (
	SignatureAnnotationSignedName = "acorn.io/signed-name" // If an image was signed by `acorn image sign foo/bar:v1`, this annotation should be set to `foo/bar:v1` (the payload usually only includes the image digest)
)

func GetDefaultSignatureAnnotations(imageName string) map[string]interface{} {
	return map[string]interface{}{
		SignatureAnnotationSignedName: imageName,
	}
}
