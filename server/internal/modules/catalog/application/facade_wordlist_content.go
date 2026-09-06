package application

import (
	"context"
	"errors"
)

// GetContent returns the content of a wordlist file.
func (service *WordlistFacade) GetContent(id int) (string, error) {
	content, err := service.cmdService.GetWordlistContent(context.Background(), id)
	if err != nil {
		return "", mapWordlistFileBoundaryError(err)
	}
	return content, nil
}

// UpdateContent updates the content of a wordlist file.
func (service *WordlistFacade) UpdateContent(id int, content string) (*Wordlist, error) {
	wordlist, err := service.cmdService.UpdateWordlistContent(context.Background(), id, content)
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			return nil, ErrFileNotFound
		}
		if errors.Is(err, ErrLineTooLong) {
			return nil, ErrLineTooLong
		}
		if isRecordNotFound(err) {
			return nil, ErrWordlistNotFound
		}
		return nil, err
	}
	return wordlist, nil
}
