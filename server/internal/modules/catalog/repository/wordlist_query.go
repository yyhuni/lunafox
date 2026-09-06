package repository

import (
	"context"
	"strings"

	catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"
	model "github.com/yyhuni/lunafox/server/internal/modules/catalog/repository/persistence"
	"github.com/yyhuni/lunafox/server/internal/pkg/scope"
	"gorm.io/gorm"
)

var wordlistFilterMapping = scope.NormalizeFilterMapping(scope.FilterMapping{
	"fileName":    {Column: "file_name"},
	"description": {Column: "description"},
	"tags":        {Column: "tags", IsJSONBStringArray: true},
	"lineCount":   {Column: "line_count", IsNumeric: true},
	"fileSize":    {Column: "file_size", IsNumeric: true},
	"fileHash":    {Column: "file_hash"},
})

// ExistsByFileName checks whether an uploaded file name already exists.
func (r *WordlistRepository) ExistsByFileName(fileName string, excludeID ...int) (bool, error) {
	var count int64
	query := r.db.Model(&model.Wordlist{}).Where("file_name = ?", fileName)
	if len(excludeID) > 0 {
		query = query.Where("id != ?", excludeID[0])
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetByIDContext reads a wordlist under the caller's execution budget.
func (r *WordlistRepository) GetByIDContext(ctx context.Context, id int) (*catalogdomain.Wordlist, error) {
	var wordlist model.Wordlist
	if err := r.db.WithContext(ctx).First(&wordlist, id).Error; err != nil {
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

// List returns paginated wordlists.
func (r *WordlistRepository) List(page, pageSize int, filter, orderBy string) ([]catalogdomain.Wordlist, int64, error) {
	return r.ListContext(context.Background(), page, pageSize, filter, orderBy)
}

// ListContext preserves the caller-owned cancellation and deadline.
func (r *WordlistRepository) ListContext(ctx context.Context, page, pageSize int, filter, orderBy string) ([]catalogdomain.Wordlist, int64, error) {
	var wordlists []model.Wordlist
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&model.Wordlist{})
	if filter != "" {
		baseQuery = applyWordlistFilter(baseQuery, filter)
	}

	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := baseQuery.Scopes(scope.WithPagination(page, pageSize))
	query = applyWordlistOrder(query, orderBy)
	if err := query.Find(&wordlists).Error; err != nil {
		return nil, 0, err
	}

	return wordlistModelListToDomain(wordlists), total, nil
}

func (r *WordlistRepository) ListTagSummaries(page, pageSize int, filter string) ([]catalogdomain.WordlistTagSummary, int64, error) {
	tags, err := r.collectTagCounts(filter)
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(tags))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(tags) {
		return []catalogdomain.WordlistTagSummary{}, total, nil
	}
	end := start + pageSize
	if end > len(tags) {
		end = len(tags)
	}
	return tags[start:end], total, nil
}

func (r *WordlistRepository) collectTagCounts(filter string) ([]catalogdomain.WordlistTagSummary, error) {
	type row struct {
		DisplayName   string
		WordlistCount int64
	}
	var rows []row
	query := r.db.Table("wordlist, jsonb_array_elements_text(tags) AS tag").
		Select("tag AS display_name, COUNT(*) AS wordlist_count").
		Where("tag <> ''").
		Group("tag").
		Order("tag ASC")
	if strings.TrimSpace(filter) != "" {
		query = query.Where("tag ILIKE ?", "%"+strings.TrimSpace(filter)+"%")
	}
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	summaries := make([]catalogdomain.WordlistTagSummary, 0, len(rows))
	for _, item := range rows {
		summaries = append(summaries, catalogdomain.WordlistTagSummary{DisplayName: item.DisplayName, WordlistCount: item.WordlistCount})
	}
	return summaries, nil
}

func applyWordlistFilter(db *gorm.DB, filter string) *gorm.DB {
	trimmed := strings.TrimSpace(filter)
	if trimmed == "" {
		return db
	}
	return db.Scopes(scope.WithFilterDefault(trimmed, wordlistFilterMapping, "fileName"))
}

func parseExactTagFilter(filter string) (string, bool) {
	for _, prefix := range []string{`tags=="`, `tags="`} {
		cleanPrefix := strings.ReplaceAll(prefix, `\`, ``)
		if strings.HasPrefix(filter, cleanPrefix) && strings.HasSuffix(filter, `"`) {
			return strings.TrimSuffix(strings.TrimPrefix(filter, cleanPrefix), `"`), true
		}
	}
	return "", false
}

func applyWordlistOrder(db *gorm.DB, orderBy string) *gorm.DB {
	switch strings.TrimSpace(orderBy) {
	case "fileName", "fileName asc":
		return db.Order("file_name ASC").Order("id ASC")
	case "fileName desc":
		return db.Order("file_name DESC").Order("id DESC")
	case "lineCount", "lineCount asc":
		return db.Order("line_count ASC").Order("id ASC")
	case "lineCount desc":
		return db.Order("line_count DESC").Order("id DESC")
	case "fileSize", "fileSize asc":
		return db.Order("file_size ASC").Order("id ASC")
	case "fileSize desc":
		return db.Order("file_size DESC").Order("id DESC")
	case "updatedAt", "updatedAt desc", "":
		return db.Order("updated_at DESC").Order("id DESC")
	case "updatedAt asc":
		return db.Order("updated_at ASC").Order("id ASC")
	default:
		return db.Where("1 = 0")
	}
}

// ListAll returns all wordlists without pagination.
func (r *WordlistRepository) ListAll() ([]catalogdomain.Wordlist, error) {
	return r.ListAllContext(context.Background())
}

// ListAllContext preserves the caller-owned cancellation and deadline.
func (r *WordlistRepository) ListAllContext(ctx context.Context) ([]catalogdomain.Wordlist, error) {
	var wordlists []model.Wordlist
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&wordlists).Error; err != nil {
		return nil, err
	}
	return wordlistModelListToDomain(wordlists), nil
}
