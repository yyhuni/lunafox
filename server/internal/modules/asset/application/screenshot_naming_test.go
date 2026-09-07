package application

import (
	"testing"
)

func TestScreenshotFacadeExposesTargetScopedListMethod(t *testing.T) {
	facade := &ScreenshotFacade{}
	_ = facade.ListByTarget
}

func TestScreenshotQueryServiceExposesTargetScopedListMethod(t *testing.T) {
	service := &ScreenshotQueryService{}
	_ = service.ListByTarget
}
