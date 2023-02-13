package strategy

type Destroyer interface {
	Destroy()
}

func NewDestroyAdapter(destroy Destroyer) *DestroyAdapter {
	return &DestroyAdapter{
		Destroyer: destroy,
	}
}

type DestroyAdapter struct {
	Destroyer Destroyer
}

func (d *DestroyAdapter) Destroy() {
	if d != nil && d.Destroyer != nil {
		d.Destroyer.Destroy()
	}
}
