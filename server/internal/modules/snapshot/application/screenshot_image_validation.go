package application

import contractresults "github.com/yyhuni/lunafox/contracts/results"

// isValidScreenshotImage is the single Server admission check for the
// persistent screenshot product. Only the result-contract WebP is accepted.
func isValidScreenshotImage(imageData []byte) bool {
	return contractresults.ValidateScreenshotWebP(imageData) == nil
}
