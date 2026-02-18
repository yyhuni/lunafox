package application

import (
	"context"
	"errors"
	"io"
)

// Create creates a new wordlist with file upload.
func (service *WordlistFacade) Create(name, description, filename string, fileContent io.Reader) (*Wordlist, error) {
	wordlist, err := service.cmdService.CreateWordlist(context.Background(), name, description, filename, fileContent)
	if err != nil {
		if errors.Is(err, ErrWordlistExists) {
			return nil, ErrWordlistExists
		}
		if errors.Is(err, ErrEmptyName) {
			return nil, ErrEmptyName
		}
		if errors.Is(err, ErrNameTooLong) {
			return nil, ErrNameTooLong
		}
		if errors.Is(err, ErrInvalidName) {
			return nil, ErrInvalidName
		}
		if errors.Is(err, ErrInvalidFileType) {
			return nil, ErrInvalidFileType
		}
		if errors.Is(err, ErrLineTooLong) {
			return nil, ErrLineTooLong
		}
		return nil, err
	}

	return wordlist, nil
}
