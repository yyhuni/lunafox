package application

import (
	"context"
	"errors"
	"github.com/yyhuni/lunafox/server/internal/pkg/dberrors"
)

// GetContent returns the content of a wordlist file.
func (service *WordlistFacade) GetContent(id int) (string, error) {
	content, err := service.cmdService.GetWordlistContent(context.Background(), id)
	if err != nil {
		if errors.Is(err, ErrWordlistNotFound) {
			return "", ErrWordlistNotFound
		}
		if errors.Is(err, ErrFileNotFound) {
			return "", ErrFileNotFound
		}
		if dberrors.IsRecordNotFound(err) {
			return "", ErrWordlistNotFound
		}
		return "", err
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
		if dberrors.IsRecordNotFound(err) {
			return nil, ErrWordlistNotFound
		}
		return nil, err
	}
	return wordlist, nil
}
