module github.com/yyhuni/lunafox/engines/subdomain_discovery

go 1.26.0

require (
	github.com/d3mondev/puredns/v2 v2.1.2-0.20260223162428-46bd4c963ec2
	github.com/d3mondev/resolvermt v0.3.2
	github.com/stretchr/testify v1.11.1
	github.com/yyhuni/lunafox/engine-go v0.0.0
	golang.org/x/net v0.56.0
)

require (
	github.com/andres-erbsen/clock v0.0.0-20160526145045-9e14626cd129 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/miekg/dns v1.1.53 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/ratelimit v0.2.0 // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/grpc v1.82.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/yyhuni/lunafox/engine-go => ../../../engine-go
