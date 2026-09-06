package application

import "context"

// MutableObservationMaterializationCoordinator owns the same-database
// Snapshot, Asset, and Scan-summary transaction. Its implementation belongs
// to infrastructure so application code never receives a GORM handle.
type MutableObservationMaterializationCoordinator interface {
	Materialize(context.Context, int, int, func(context.Context) error) error
}
