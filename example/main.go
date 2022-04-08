package main

import (
	"fmt"

	"github.com/joamaki/genbus"
	"github.com/joamaki/genbus/example/pkg/sbbtraffic"
	"github.com/joamaki/genbus/example/pkg/store"
	"go.uber.org/fx"
)

func printStats(up store.StoreUpdatedEvent) error {
	fmt.Println(up)
	return nil
}

func subscribe(builder *genbus.Builder, st *sbbtraffic.SBBTraffic, storeAPI store.API) {
	genbus.Subscribe(builder, "Print arrivals",
		func(arr *sbbtraffic.ArrivalEvent) error {
			fmt.Println(arr)
			fmt.Println(storeAPI.GetArrivalTimes(arr.Line))
			fmt.Println("---")
			return nil
		})

	genbus.Subscribe(builder, "Print departures",
		func(dep *sbbtraffic.DepartureEvent) error {
			fmt.Println(dep)
			fmt.Println(storeAPI.GetDepartureTimes(dep.Line))
			fmt.Println("---")
			return nil
		})

	genbus.Subscribe(builder, "Print store stats", printStats)
}

func main() {
	app := fx.New(
		fx.Provide(
			genbus.NewBuilder,
			store.NewStore,
			sbbtraffic.NewSBBTraffic,
		),
		fx.Invoke(
			subscribe,
			func(builder *genbus.Builder) (err error) {
				bus, err := builder.Build()
				bus.PrintGraph()
				return
			},
		),
	)

	app.Run()
}
