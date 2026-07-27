package toolgateway

import "context"

func (s *Service) List(ctx context.Context) ([]Definition, error) {
	if s == nil || s.registry == nil {
		return nil, ErrToolNotFound
	}
	return s.registry.List(ctx)
}
