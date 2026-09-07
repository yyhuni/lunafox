module github.com/yyhuni/lunafox/tools/engine-oci-publish

go 1.26.0

require (
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.1
	github.com/yyhuni/lunafox/contracts v0.0.0
	oras.land/oras-go/v2 v2.6.2
)

require golang.org/x/sync v0.22.0 // indirect

replace github.com/yyhuni/lunafox/contracts => ../../contracts
