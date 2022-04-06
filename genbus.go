// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

package genbus

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

type Event interface {
	fmt.Stringer
}

type eventTypeId uint64

// eventToTypeId extracts event's type identifier
func eventToTypeId(ev Event) eventTypeId {
	type fatPtr struct {
		rtype uintptr
		w     uintptr
	}
	return eventTypeId((*fatPtr)(unsafe.Pointer(&ev)).rtype)
}

func eventToTypeName(x interface{}) string {
	return reflect.TypeOf(x).String()
	/*typ := reflect.TypeOf(x)
	return typ.PkgPath() + "." + typ.Name()*/
}

// EventBus provides a publish-subscribe service for broadcasting events from a subsystem.
type EventBus struct {
	sync.RWMutex

	pubs map[eventTypeId]*publisher
	subs map[eventTypeId][]*subscriber

	types map[eventTypeId]string
}

func NewEventBus() *EventBus {
	return &EventBus{
		pubs:  map[eventTypeId]*publisher{},
		subs:  map[eventTypeId][]*subscriber{},
		types: map[eventTypeId]string{},
	}
}

func (bus *EventBus) DumpSubs() {
	bus.RLock()
	defer bus.RUnlock()

	// Construct the reverse of 'subs'
	revSubs := map[*subscriber][]eventTypeId{}
	for typeId, subs := range bus.subs {
		for _, sub := range subs {
			revSubs[sub] = append(revSubs[sub], typeId)
		}
	}

	for subsys, typs := range revSubs {
		fmt.Printf("%s:\n", subsys.name)
		for _, typ := range typs {
			fmt.Printf("\t%s\n", bus.types[typ])
		}
	}
}

type publisher struct {
	name string
}

type subscriber struct {
	name    string
	handler func(ev Event) error
}

func Subscribe[E Event](bus *EventBus, name string, handler func(event E) error) error {
	bus.Lock()
	defer bus.Unlock()

	// Construct an empty event of type 'E' and extract its type id
	var e E
	typeId := eventToTypeId(e)

	if _, ok := bus.pubs[typeId]; !ok {
		return fmt.Errorf("no publisher found for type %s", eventToTypeName(e))
	}

	sub := &subscriber{
		name: name,
		// Create a handler for 'Event' from the handler for 'E'. We know
		// this is safe as it is indexed by the type id.
		handler: func(ev Event) error { return handler(ev.(E)) },
	}
	bus.subs[typeId] = append(bus.subs[typeId], sub)
	return nil
}

type PublishFn[E Event] func(ev E) error

func (bus *EventBus) publish(typeId eventTypeId, ev Event) error {
	bus.RLock()
	defer bus.RUnlock()

	subs, ok := bus.subs[typeId]
	if !ok || len(subs) == 0 {
		return fmt.Errorf("no subscribers to %s", eventToTypeName(ev))
	}

	for _, sub := range subs {
		sub.handler(ev)
	}
	return nil
}

func Register[E Event](bus *EventBus, name string) (PublishFn[E], error) {
	bus.Lock()
	defer bus.Unlock()

	var e E
	typeId := eventToTypeId(e)
	typeName := eventToTypeName(e)
	bus.pubs[typeId] = &publisher{name}
	bus.types[typeId] = typeName

	return func(ev E) error {
		return bus.publish(typeId, ev)
	}, nil
}
