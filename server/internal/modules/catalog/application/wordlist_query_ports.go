package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type WordlistQueryStore interface {
	FindAll(page, pageSize int) ([]catalogdomain.Wordlist, int64, error)
	List() ([]catalogdomain.Wordlist, error)
	GetByID(id int) (*catalogdomain.Wordlist, error)
	FindByName(name string) (*catalogdomain.Wordlist, error)
	Update(wordlist *catalogdomain.Wordlist) error
}
