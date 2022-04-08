package sbbtraffic

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/joamaki/genbus"
	opentransdata "github.com/niccantieni/golang-opentransdata"
	"go.uber.org/fx"
)

var (
	// Get your API key from https://opentransportdata.swiss/
	apiKey    = "<apikey here>"
	fakeIt    = true
	stationId = "8503000"
)

func getTriasNow(stationID string, departure bool) (trias opentransdata.Trias, err error) {
	request := opentransdata.TemplateOTDRequestNow()
	request.StopPointRef = stationID
	request.Parameters.NumberOfResults = "100"

	if !departure {
		request.Parameters.StopEventType = "arrival"
	}

	data, err := opentransdata.CreateRequest(apiKey, request)
	if err != nil {
		return trias, err
	}
	trias, err = opentransdata.ParseXML(data)
	if err != nil {
		return trias, err
	}

	return trias, err
}

type DepartureEvent struct {
	Line          string
	DepartureTime string
}

func (dep *DepartureEvent) Key() string {
	return dep.Line + "/" + dep.DepartureTime
}

func (dep *DepartureEvent) String() string {
	return fmt.Sprintf("%s: Departure of %s", dep.DepartureTime, dep.Line)
}

type ArrivalEvent struct {
	Line        string
	ArrivalTime string
}

func (arr *ArrivalEvent) Key() string {
	return arr.Line + "/" + arr.ArrivalTime
}

func (arr *ArrivalEvent) String() string {
	return fmt.Sprintf("%s: Arrival of %s", arr.ArrivalTime, arr.Line)
}

type SBBTraffic struct {
	ctx                 context.Context
	publishedDepartures map[string]bool
	publishedArrivals   map[string]bool

	publishDeparture genbus.PublishFn[*DepartureEvent]
	publishArrival   genbus.PublishFn[*ArrivalEvent]
}

func (st *SBBTraffic) pull() {
	trias, _ := getTriasNow(stationId, true)
	for _, dep := range trias.ServiceDelivery.DeliveryPayload.StopEventResponse.StopEventResult {
		stop := dep.StopEvent
		ev := &DepartureEvent{
			Line:          stop.Service.PublishedLineName.Text,
			DepartureTime: stop.ThisCall.CallAtStop.ServiceDeparture.TimetabledTime.String,
		}
		if !st.publishedDepartures[ev.Key()] {
			st.publishDeparture(ev)
			st.publishedDepartures[ev.Key()] = true
		}
	}

	trias, _ = getTriasNow(stationId, false)
	for _, arr := range trias.ServiceDelivery.DeliveryPayload.StopEventResponse.StopEventResult {
		stop := arr.StopEvent
		ev := &ArrivalEvent{
			stop.Service.PublishedLineName.Text,
			stop.ThisCall.CallAtStop.ServiceArrival.TimetabledTime.String,
		}
		if !st.publishedArrivals[ev.Key()] {
			st.publishArrival(ev)
			st.publishedArrivals[ev.Key()] = true
		}
	}
}

func fakeDeparture() *DepartureEvent {
	line := fmt.Sprintf("S%d", rand.Int31n(30))
	return &DepartureEvent{
		Line:          line,
		DepartureTime: time.Now().Format(time.RFC3339),
	}
}

func fakeArrival() *ArrivalEvent {
	line := fmt.Sprintf("S%d", rand.Int31n(30))
	return &ArrivalEvent{
		Line:        line,
		ArrivalTime: time.Now().Format(time.RFC3339),
	}
}

func (st *SBBTraffic) fakePull() {
	for i := 0; i < 5; i++ {
		st.publishDeparture(fakeDeparture())
		st.publishArrival(fakeArrival())
	}
}

func (st *SBBTraffic) loop() {
	for st.ctx.Err() == nil {
		if fakeIt {
			st.fakePull()
		} else {
			st.pull()
		}
		if fakeIt {
			time.Sleep(time.Second)
		} else {
			time.Sleep(time.Minute)
		}
	}
}

func NewSBBTraffic(lc fx.Lifecycle, bus *genbus.Builder) (*SBBTraffic, error) {
	ctx, cancel := context.WithCancel(context.Background())
	st := &SBBTraffic{
		ctx:                 ctx,
		publishedDepartures: make(map[string]bool),
		publishedArrivals:   make(map[string]bool),
	}
	var err error
	st.publishDeparture, err = genbus.Register[*DepartureEvent](bus, "Departing trains at Zurich HB")
	if err != nil {
		cancel()
		return nil, err
	}
	st.publishArrival, err = genbus.Register[*ArrivalEvent](bus, "Arriving trains at Zurich HB")
	if err != nil {
		cancel()
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error { go st.loop(); return nil },
		OnStop:  func(context.Context) error { cancel(); return nil },
	})

	return st, nil
}
