package application

import catalogdomain "github.com/yyhuni/lunafox/server/internal/modules/catalog/domain"

type WordlistCommandStore interface {
	GetByID(id int) (*catalogdomain.Wordlist, error)
	ExistsByName(name string) (bool, error)
	Create(wordlist *catalogdomain.Wordlist) error
	Update(wordlist *catalogdomain.Wordlist) error
	Delete(id int) error
}
