package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type ScreenshotCommandStore interface {
	BatchDelete(ids []int) (int64, error)
	BatchUpsert(screenshots []assetdomain.Screenshot) (int64, error)
}

type ScreenshotStore interface {
	ScreenshotQueryStore
	ScreenshotCommandStore
}
