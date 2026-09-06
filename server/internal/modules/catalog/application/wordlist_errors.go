package application

import (
	"errors"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
)

var (
	ErrWordlistNotFound = catalogdomain.ErrWordlistNotFound
	ErrWordlistExists   = catalogdomain.ErrWordlistExists
	ErrEmptyFileName    = catalogdomain.ErrWordlistFileNameEmpty
	ErrFileNameTooLong  = catalogdomain.ErrWordlistFileNameTooLong
	ErrInvalidFileName  = catalogdomain.ErrWordlistFileNameInvalid
	ErrFileNotFound     = catalogdomain.ErrWordlistFileNotFound
	ErrInvalidFileType  = catalogdomain.ErrWordlistInvalidFileType
	ErrLineTooLong      = catalogdomain.ErrWordlistLineTooLong

	ErrUnsupportedWordlistFilter  = errors.New("unsupported wordlist filter")
	ErrUnsupportedWordlistOrderBy = errors.New("unsupported wordlist orderBy")
	ErrInvalidWordlistPageToken   = errors.New("invalid wordlist pageToken")
)
