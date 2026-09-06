package application

import (
	"errors"
	"fmt"

	contractresults "github.com/yyhuni/lunafox/contracts/results"
)

// ErrInvalidObservedAssetURL marks a client-supplied manual batch URL that
// cannot enter the observed-asset boundary without losing its original value.
var ErrInvalidObservedAssetURL = errors.New("invalid observed asset URL")

func observedAssetURLHostForBatchCreate(index int, rawURL string) (string, error) {
	if _, err := contractresults.ValidateObservedAssetURL(rawURL); err != nil {
		return "", fmt.Errorf("%w: urls[%d]: %v", ErrInvalidObservedAssetURL, index, err)
	}
	host, err := contractresults.DeriveObservedAssetURLHost(rawURL)
	if err != nil {
		return "", fmt.Errorf("%w: urls[%d]: %v", ErrInvalidObservedAssetURL, index, err)
	}
	return host, nil
}
