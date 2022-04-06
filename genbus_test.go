package genbus

import (
	"fmt"
	"testing"
)

type EventA struct{}

func (a *EventA) String() string { return "EventA" }

type EventB struct{}

func (a *EventB) String() string { return "EventB" }

type EventC int

func (c EventC) String() string { return "EventC" }

func TestGenbus(t *testing.T) {

	bus := NewEventBus()

	pubA, err := Register[*EventA](bus, "source of EventA")
	if err != nil {
		t.Fatal(err)
	}

	// Test consuming of EventA
	countA := 0
	_, err = Subscribe(bus, "print EventA",
		func(ev *EventA) error {
			countA++
			fmt.Printf("evA: %s\n", ev)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}

	pubA(&EventA{})
	pubA(&EventA{})

	if countA != 2 {
		t.Fatalf("expected to see 2 events, got %d", countA)
	}

	// Test subscribing to events that are not registered
	_, err = Subscribe(bus, "unknown", func(ev *EventB) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected subscribing to unregistered event to fail")
	}

	// Test with another type
	pubB, err := Register[*EventB](bus, "source of EventB")
	if err != nil {
		t.Fatal(err)
	}

	countB := 0
	_, err = Subscribe(bus, "print EventB",
		func(ev *EventB) error {
			fmt.Printf("evB: %s\n", ev)
			countB++
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	pubB(&EventB{})

	if countB != 1 {
		t.Fatalf("expected to see 1 event, got %d", countB)
	}

	// Test with a non-pointer type
	pubC, err := Register[EventC](bus, "source of EventC")
	if err != nil {
		t.Fatal(err)
	}

	countC := 0
	unsubC, err := Subscribe(bus, "print EventC",
		func(ev EventC) error {
			countC++
			fmt.Printf("evC: %s\n", ev)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	pubC(EventC(123))
	pubC(EventC(123))
	pubC(EventC(123))
	unsubC()
	pubC(EventC(123))
	pubC(EventC(123))

	if countC != 3 {
		t.Fatalf("expected to see 3 events, got %d", countC)
	}

}

func BenchmarkGenbus(b *testing.B) {
	bus := NewEventBus()

	pubA, err := Register[*EventA](bus, "source of EventA")
	if err != nil {
		b.Fatal(err)
	}

	count := 0
	_, err = Subscribe(bus, "count EventAs",
		func(ev *EventA) error {
			count++
			return nil
		})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pubA(&EventA{})
	}
	if count != b.N {
		b.Fatalf("missed events, expected %d, got %d", b.N, count)
	}
}
