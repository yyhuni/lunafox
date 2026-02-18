package handler

import (
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

// WordlistHandler handles wordlist API requests.
type WordlistHandler struct {
	svc *service.WordlistFacade
}

// NewWordlistHandler creates a new wordlist handler.
func NewWordlistHandler(svc *service.WordlistFacade) *WordlistHandler {
	return &WordlistHandler{svc: svc}
}

func toWordlistOutput(wordlist *service.Wordlist) dto.WordlistResponse {
	return dto.WordlistResponse{
		ID:          wordlist.ID,
		Name:        wordlist.Name,
		Description: wordlist.Description,
		FilePath:    wordlist.FilePath,
		FileSize:    wordlist.FileSize,
		LineCount:   wordlist.LineCount,
		FileHash:    wordlist.FileHash,
		CreatedAt:   timeutil.ToUTC(wordlist.CreatedAt),
		UpdatedAt:   timeutil.ToUTC(wordlist.UpdatedAt),
	}
}
