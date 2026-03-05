package preset

import "errors"

// ErrPresetNotFound is returned when a preset with the given ID is not found.
var ErrPresetNotFound = errors.New("preset not found")

// Service provides business logic for preset engines.
type Service struct {
	loader *Loader
}

// NewService creates a new preset Service.
func NewService(loader *Loader) *Service {
	return &Service{loader: loader}
}

// List returns all available presets.
func (s *Service) List() []Preset {
	return s.loader.List()
}

// GetByID returns a preset by its ID.
func (s *Service) GetByID(id string) (*Preset, error) {
	preset := s.loader.GetByID(id)
	if preset == nil {
		return nil, ErrPresetNotFound
	}
	return preset, nil
}
