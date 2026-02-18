package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/timeutil"
)

func wordlistModelToDomain(wordlist *model.Wordlist) *catalogdomain.Wordlist {
	if wordlist == nil {
		return nil
	}
	return &catalogdomain.Wordlist{
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

func wordlistDomainToModel(wordlist *catalogdomain.Wordlist) *model.Wordlist {
	if wordlist == nil {
		return nil
	}
	return &model.Wordlist{
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

func wordlistModelListToDomain(wordlists []model.Wordlist) []catalogdomain.Wordlist {
	results := make([]catalogdomain.Wordlist, 0, len(wordlists))
	for index := range wordlists {
		results = append(results, *wordlistModelToDomain(&wordlists[index]))
	}
	return results
}
