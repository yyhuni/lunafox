package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type ScreenshotQueryStore interface {
	FindByTargetID(targetID int, page, pageSize int, filter string) ([]assetdomain.Screenshot, int64, error)
	GetByID(id int) (*assetdomain.Screenshot, error)
}
