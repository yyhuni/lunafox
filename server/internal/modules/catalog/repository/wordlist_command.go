package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
)

// Create creates a new wordlist.
func (r *WordlistRepository) Create(wordlist *catalogdomain.Wordlist) error {
	modelWordlist := wordlistDomainToModel(wordlist)
	if err := r.db.Create(modelWordlist).Error; err != nil {
		return err
	}
	if wordlist != nil {
		*wordlist = *wordlistModelToDomain(modelWordlist)
	}
	return nil
}

// Update updates a wordlist.
func (r *WordlistRepository) Update(wordlist *catalogdomain.Wordlist) error {
	return r.db.Save(wordlistDomainToModel(wordlist)).Error
}

// Delete deletes a wordlist by ID.
func (r *WordlistRepository) Delete(id int) error {
	return r.db.Delete(&model.Wordlist{}, id).Error
}
