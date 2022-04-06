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
}

// EventBus provides a publish-subscribe service for broadcasting events from a subsystem.
type EventBus struct {
	sync.Mutex

	pubs  map[eventTypeId]*publisher
	types map[eventTypeId]string
}

func NewEventBus() *EventBus {
	return &EventBus{
		pubs:  map[eventTypeId]*publisher{},
		types: map[eventTypeId]string{},
	}
}

func (bus *EventBus) DumpSubs() {
	bus.Lock()
	defer bus.Unlock()

	// Construct the reverse of 'subs'
	revSubs := map[*subscriber][]eventTypeId{}
	for typeId, pub := range bus.pubs {
		pub.RLock()
		for _, sub := range pub.subs {
			revSubs[sub] = append(revSubs[sub], typeId)
		}
		pub.RUnlock()
	}

	for sub, typs := range revSubs {
		fmt.Printf("%s:\n", sub.name)
		for _, typ := range typs {
			fmt.Printf("\t%s\n", bus.types[typ])
		}
	}
}

type publisher struct {
	sync.RWMutex
	name string
	subs []*subscriber
}

type subscriber struct {
	name    string
	handler func(ev Event) error
}

type UnsubscribeFn func()

func Subscribe[E Event](bus *EventBus, name string, handler func(event E) error) (UnsubscribeFn, error) {
	bus.Lock()
	defer bus.Unlock()

	// Construct an empty event of type 'E' and extract its type id
	var e E
	typeId := eventToTypeId(e)

	pub, ok := bus.pubs[typeId]
	if !ok {
		return nil, fmt.Errorf("no publisher found for type %s", eventToTypeName(e))
	}
	pub.Lock()
	defer pub.Unlock()

	sub := &subscriber{
		name: name,
		// Create a handler for 'Event' from the handler for 'E'. We know
		// this is safe as it is indexed by the type id.
		handler: func(ev Event) error { return handler(ev.(E)) },
	}
	pub.subs = append(pub.subs, sub)

	unsub := func() {
		pub.Lock()
		defer pub.Unlock()
		for i, s := range pub.subs {
			if s == sub {
				pub.subs = deleteUnordered(pub.subs, i)
				break
			}
		}
	}

	return unsub, nil
}

type PublishFn[E Event] func(ev E) error

func (pub *publisher) publish(typeId eventTypeId, ev Event) error {
	pub.RLock()
	for _, sub := range pub.subs {
		sub.handler(ev)
	}
	pub.RUnlock()
	return nil
}

func Register[E Event](bus *EventBus, name string) (PublishFn[E], error) {
	bus.Lock()
	defer bus.Unlock()

	var e E
	typeId := eventToTypeId(e)
	typeName := eventToTypeName(e)
	pub := &publisher{
		name: name,
		subs: []*subscriber{},
	}
	bus.pubs[typeId] = pub
	bus.types[typeId] = typeName

	return func(ev E) error { return pub.publish(typeId, ev) }, nil
}

func deleteUnordered[T any](slice []T, index int) []T {
	var empty T
	slice[index] = slice[len(slice)-1]
	slice[len(slice)-1] = empty
	return slice[:len(slice)-1]
}
