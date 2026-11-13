module example.com/conformance

go 1.27

require (
	github.com/ianflanagan1/go-shared-code v0.0.0
	go.uber.org/fx v0.0.0
)

replace (
	github.com/ianflanagan1/go-shared-code => ./stub/go-shared-code
	go.uber.org/fx => ./stub/fx
)
