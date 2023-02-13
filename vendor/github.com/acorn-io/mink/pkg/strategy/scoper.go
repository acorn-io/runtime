package strategy

type Scoper interface {
	NamespaceScoped() bool
}

type ScoperAdapter struct {
	strategy Newer
}

func NewScoper(strategy Newer) *ScoperAdapter {
	return &ScoperAdapter{
		strategy: strategy,
	}
}

func (s *ScoperAdapter) NamespaceScoped() bool {
	if s != nil {
		if o, ok := s.strategy.(NamespaceScoper); ok {
			return o.NamespaceScoped()
		}
		if o, ok := s.strategy.New().(NamespaceScoper); ok {
			return o.NamespaceScoped()
		}
	}
	return true
}
