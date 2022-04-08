package genbus

import (
	"testing"
)

type EventA struct{}

func (a *EventA) String() string { return "EventA" }

type EventB struct{}

func (a *EventB) String() string { return "EventB" }

type EventC int

func (c EventC) String() string { return "EventC" }

func TestGenbus(t *testing.T) {

	builder := NewBuilder()

	pubA, err := Register[*EventA](builder, "source of EventA")
	if err != nil {
		t.Fatal(err)
	}

	// Publishing before the bus is ready fails
	err = pubA(&EventA{})
	if err == nil {
		t.Fatalf("expected publishing before bus is built to fail")
	}

	// Test consuming of EventA
	countA := 0
	unsub := Subscribe(builder, "count EventA",
		func(ev *EventA) error {
			countA++
			return nil
		})

	_, err = builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	err = pubA(&EventA{})
	if err != nil {
		t.Fatalf("expected publishing to succeed, got %s", err)
	}
	pubA(&EventA{})

	unsub()

	err = pubA(&EventA{})
	if err != nil {
		t.Fatalf("expected publishing to succeed even without subscribers, but got %s", err)
	}

	if countA != 2 {
		t.Fatalf("expected to see 2 events, got %d", countA)
	}

}

func TestGenbusUnknownEvent(t *testing.T) {
	builder := NewBuilder()

	// Test subscribing to events that are not registered
	_ = Subscribe(builder, "unknown", func(ev *EventB) error {
		return nil
	})

	_, err := builder.Build()
	if err == nil {
		t.Fatalf("expected subscribing to unregistered event to fail")
	}
}

func TestGenbusNonPointer(t *testing.T) {
	builder := NewBuilder()

	// Test with a non-pointer type
	pubC, err := Register[EventC](builder, "source of EventC")
	if err != nil {
		t.Fatal(err)
	}

	countC := 0
	unsubC := Subscribe(builder, "count EventC",
		func(ev EventC) error {
			countC++
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build()
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

func TestGenbusOutOfOrder(t *testing.T) {
	builder := NewBuilder()

	// First subscribe to the event
	countA := 0
	unsub := Subscribe(builder, "count As",
		func(ev *EventA) error { countA++; return nil })

	// Then register the publisher
	pubA, err := Register[*EventA](builder, "source of EventA")
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build()
	if err != nil {
		t.Fatal(err)
	}

	pubA(&EventA{})
	pubA(&EventA{})
	unsub()
	pubA(&EventA{})

	if countA != 2 {
		t.Fatalf("expected to see 2 events, got %d", countA)
	}
}

func BenchmarkGenbus(b *testing.B) {
	builder := NewBuilder()

	pubA, err := Register[*EventA](builder, "source of EventA")
	if err != nil {
		b.Fatal(err)
	}

	count := 0
	_ = Subscribe(builder, "count EventAs",
		func(ev *EventA) error {
			count++
			return nil
		})
	if err != nil {
		b.Fatal(err)
	}

	_, err = builder.Build()
	if err != nil {
		b.Fatal(err)
	}

	ev := &EventA{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pubA(ev)
	}
	if count != b.N {
		b.Fatalf("missed events, expected %d, got %d", b.N, count)
	}
}

// BenchmarkFuncCall gives a baseline performance to which to compare
// BenchmarkGenbus to.
func BenchmarkFuncCall(b *testing.B) {
	count := 0

	var cb func(ev *EventA) error
	if b.N > 0 {
		// Hack to stop Go from inlining
		cb = func(ev *EventA) error {
			count++
			return nil
		}
	}

	ev := &EventA{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb(ev)
	}
	if count != b.N {
		b.Fatalf("missed events, expected %d, got %d", b.N, count)
	}
}
