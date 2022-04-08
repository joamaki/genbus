module github.com/joamaki/genbus/example

go 1.18

require (
	github.com/joamaki/genbus v0.0.0
	github.com/niccantieni/golang-opentransdata v0.0.0-20210214171703-bbc86313719b
	go.uber.org/fx v1.17.1
)

require (
	go.uber.org/atomic v1.6.0 // indirect
	go.uber.org/dig v1.14.0 // indirect
	go.uber.org/multierr v1.5.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/sys v0.0.0-20210903071746-97244b99971b // indirect
)

replace github.com/joamaki/genbus => ../
