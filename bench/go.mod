module aoanima.ru/bench

go 1.21.2

replace aoanima.ru/ConnQuic => ../ConnQuic

replace aoanima.ru/Logger => ../Logger

require (
	aoanima.ru/Logger v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.18.0 // indirect
)

require (
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/onsi/ginkgo/v2 v2.9.5 // indirect
	github.com/quic-go/qtls-go1-20 v0.4.1 // indirect
	github.com/quic-go/quic-go v0.40.1 // indirect
	go.uber.org/mock v0.3.0 // indirect
	golang.org/x/crypto v0.15.0 // indirect
	golang.org/x/exp v0.0.0-20221205204356-47842c84f3db // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/tools v0.9.1 // indirect
)
