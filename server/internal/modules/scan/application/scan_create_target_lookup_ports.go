package application

type ScanCreateTargetLookup interface {
	GetTargetRefByID(id int) (*TargetRef, error)
}
