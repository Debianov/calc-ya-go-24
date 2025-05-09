module github.com/Debianov/calc-ya-go-24/backend/agent

go 1.24.0

replace github.com/Debianov/calc-ya-go-24 v0.0.0-20250302045807-432e7a102e57 => ../..

require (
	github.com/Debianov/calc-ya-go-24 v0.0.0-20250302045807-432e7a102e57
	github.com/stretchr/testify v1.10.0
	google.golang.org/grpc v1.72.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
