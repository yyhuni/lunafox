// Package nucleipocwiring assembles the Nuclei POC repository at the
// composition root while keeping the terminal notification sink explicit.
package nucleipocwiring

import (
	"github.com/yyhuni/lunafox/server/internal/modules/notification/repository"
	nucleipocrepo "github.com/yyhuni/lunafox/server/internal/modules/nucleipoc/repository"
	"gorm.io/gorm"
)

// NewNucleiPOCRepository wires the catalog repository with its optional
// transaction-owned notification producer bridge.
func NewNucleiPOCRepository(db *gorm.DB, producer *repository.ProducerWriter) *nucleipocrepo.NucleiPOCRepository {
	if db == nil {
		panic("nuclei POC wiring database is required")
	}
	if producer == nil {
		panic("nuclei POC wiring notification producer is required")
	}
	return nucleipocrepo.NewNucleiPOCRepository(db, producer)
}
