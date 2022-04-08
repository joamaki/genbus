package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/joamaki/genbus"
	"github.com/joamaki/genbus/example/pkg/sbbtraffic"
	"go.uber.org/fx"
)

type store struct {
	sync.RWMutex

	totalArrivals, totalDepartures int
	arrivals                       map[string][]*sbbtraffic.ArrivalEvent
	departures                     map[string][]*sbbtraffic.DepartureEvent

	publishUpdate genbus.PublishFn[StoreUpdatedEvent]
}

type StoreUpdatedEvent struct {
	NumArrivals   int
	NumDepartures int
}

func (u StoreUpdatedEvent) String() string {
	return fmt.Sprintf("store updated: arrivals=%d, departures=%d", u.NumArrivals, u.NumDepartures)
}

type API interface {
	GetArrivalTimes(line string) []string
	GetDepartureTimes(line string) []string
}

func NewStore(lc fx.Lifecycle, bus *genbus.Builder) (API, error) {
	s := &store{
		arrivals:   map[string][]*sbbtraffic.ArrivalEvent{},
		departures: map[string][]*sbbtraffic.DepartureEvent{},
	}

	var err error
	s.publishUpdate, err = genbus.Register[StoreUpdatedEvent](bus, "Store updated")
	if err != nil {
		return nil, err
	}

	unsubArrival := genbus.Subscribe(bus, "Store arrivals", s.storeArrival)
	unsubDeparture := genbus.Subscribe(bus, "Store departures", s.storeDeparture)

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return nil
		},
		OnStop: func(context.Context) error {
			unsubArrival()
			unsubDeparture()
			return nil
		},
	})

	return s, nil
}

func (s *store) updated() {
	s.publishUpdate(StoreUpdatedEvent{s.totalArrivals, s.totalDepartures})
}

func (s *store) storeArrival(arr *sbbtraffic.ArrivalEvent) error {
	s.Lock()
	defer s.Unlock()
	s.arrivals[arr.Line] = append(s.arrivals[arr.Line], arr)
	s.totalArrivals++
	s.updated()
	return nil
}

func (s *store) storeDeparture(dep *sbbtraffic.DepartureEvent) error {
	s.Lock()
	defer s.Unlock()
	s.departures[dep.Line] = append(s.departures[dep.Line], dep)
	s.totalDepartures++
	s.updated()
	return nil
}

func (s *store) GetArrivalTimes(line string) []string {
	s.RLock()
	defer s.RUnlock()

	ts := []string{}
	for _, arr := range s.arrivals[line] {
		ts = append(ts, arr.ArrivalTime)
	}
	return ts
}

func (s *store) GetDepartureTimes(line string) []string {
	s.RLock()
	defer s.RUnlock()

	ts := []string{}
	for _, dep := range s.departures[line] {
		ts = append(ts, dep.DepartureTime)
	}
	return ts
}
