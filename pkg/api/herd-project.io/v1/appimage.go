package v1

type AppImage struct {
	ID        string
	Herdfile  []byte
	ImageData ImageData
}
