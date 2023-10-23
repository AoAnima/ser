module aoanima.ru/mainServer

go 1.21.1

replace aoanima.ru/logger => ../logger

require (
	aoanima.ru/logger v0.0.0-00010101000000-000000000000
	github.com/dgryski/go-metro v0.0.0-20211217172704-adc40b04c140
	github.com/google/uuid v1.3.1
	github.com/json-iterator/go v1.1.12
)

require (
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
)
