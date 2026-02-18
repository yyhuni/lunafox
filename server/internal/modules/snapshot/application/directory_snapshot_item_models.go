package application

type DirectorySnapshotItem struct {
	URL           string
	Status        *int
	ContentLength *int
	ContentType   string
	Duration      *int
}

type DirectoryAssetUpsertItem struct {
	URL           string
	Status        *int
	ContentLength *int
	ContentType   string
	Duration      *int
}
