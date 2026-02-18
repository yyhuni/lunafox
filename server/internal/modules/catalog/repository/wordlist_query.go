package repository

import (
	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	"github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
)

// ExistsByName checks if a wordlist with the given name exists.
func (r *WordlistRepository) ExistsByName(name string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.Wordlist{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// FindByName finds a wordlist by name.
func (r *WordlistRepository) FindByName(name string) (*catalogdomain.Wordlist, error) {
	var wordlist model.Wordlist
	if err := r.db.Where("name = ?", name).First(&wordlist).Error; err != nil {
		return nil, err
	}
	return wordlistModelToDomain(&wordlist), nil
}

// GetByID finds a wordlist by ID.
func (r *WordlistRepository) GetByID(id int) (*catalogdomain.Wordlist, error) {
	var wordlist model.Wordlist
	if err := r.db.First(&wordlist, id).Error; err != nil {
		return nil, err
	}
	return wordlistModelToDomain(&wordlist), nil
}

// FindAll returns paginated wordlists.
func (r *WordlistRepository) FindAll(page, pageSize int) ([]catalogdomain.Wordlist, int64, error) {
	var wordlists []model.Wordlist
	var total int64

	if err := r.db.Model(&model.Wordlist{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&wordlists).Error; err != nil {
		return nil, 0, err
	}

	return wordlistModelListToDomain(wordlists), total, nil
}

// List returns all wordlists (no pagination).
func (r *WordlistRepository) List() ([]catalogdomain.Wordlist, error) {
	var wordlists []model.Wordlist
	if err := r.db.Order("created_at DESC").Find(&wordlists).Error; err != nil {
		return nil, err
	}
	return wordlistModelListToDomain(wordlists), nil
}
