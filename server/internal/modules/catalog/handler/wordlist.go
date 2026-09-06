package handler

import (
	service "github.com/yyhuni/lunafox/server/internal/modules/catalog/application"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/dto"
	"github.com/yyhuni/lunafox/server/internal/modules/httpdto"
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
		Name:        httpdto.WordlistName(wordlist.ID),
		FileName:    wordlist.FileName,
		Description: wordlist.Description,
		Tags:        append([]string(nil), wordlist.Tags...),
		FilePath:    wordlist.FilePath,
		FileSize:    wordlist.FileSize,
		LineCount:   wordlist.LineCount,
		FileHash:    wordlist.FileHash,
		CreatedAt:   timeutil.ToUTC(wordlist.CreatedAt),
		UpdatedAt:   timeutil.ToUTC(wordlist.UpdatedAt),
	}
}

func toWordlistTextOutput(id int, content string, wordlist *service.Wordlist) dto.WordlistContentResponse {
	return dto.WordlistContentResponse{
		Name:       httpdto.WordlistTextName(id),
		Content:    content,
		UpdateTime: timeutil.ToUTC(wordlist.UpdatedAt),
	}
}

func toWordlistTagSummaryOutput(summary service.WordlistTagSummary) dto.WordlistTagSummaryResponse {
	return dto.WordlistTagSummaryResponse{
		Name:          httpdto.WordlistTagName(summary.DisplayName),
		DisplayName:   summary.DisplayName,
		WordlistCount: summary.WordlistCount,
	}
}
