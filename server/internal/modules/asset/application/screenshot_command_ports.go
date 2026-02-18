package application

import assetdomain "github.com/yyhuni/lunafox/server/internal/modules/asset/domain"

type ScreenshotCommandStore interface {
	BulkDelete(ids []int) (int64, error)
	BulkUpsert(screenshots []assetdomain.Screenshot) (int64, error)
}

type ScreenshotStore interface {
	ScreenshotQueryStore
	ScreenshotCommandStore
}
