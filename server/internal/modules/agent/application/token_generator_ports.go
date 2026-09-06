package application

// TokenGenerator provides random token generation.
type TokenGenerator interface {
	GenerateHex(byteLen int) (string, error)
}
