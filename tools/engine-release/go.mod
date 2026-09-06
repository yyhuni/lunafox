module github.com/yyhuni/lunafox/tools/engine-release

go 1.26.0

require github.com/yyhuni/lunafox/contracts v0.0.0

require (
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
)

replace github.com/yyhuni/lunafox/contracts => ../../contracts
