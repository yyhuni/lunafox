package application

type DirectorySnapshotItem struct {
	URL           string
	Status        *int
	ContentLength *int64
	ContentType   string
	Duration      *int64
}

type DirectoryAssetUpsertItem struct {
	URL           string
	Status        *int
	ContentLength *int64
	ContentType   string
	Duration      *int64
}
