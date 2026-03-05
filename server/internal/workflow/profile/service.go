package profile

import "errors"

// ErrProfileNotFound is returned when a profile with the given ID is not found.
var ErrProfileNotFound = errors.New("profile not found")

// Service provides business logic for workflow profiles.
type Service struct {
	loader *Loader
}

// NewService creates a new profile Service.
func NewService(loader *Loader) *Service {
	return &Service{loader: loader}
}

// List returns all available profiles.
func (s *Service) List() []Profile {
	return s.loader.List()
}

// GetByID returns a profile by its ID.
func (s *Service) GetByID(id string) (*Profile, error) {
	profile := s.loader.GetByID(id)
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}
