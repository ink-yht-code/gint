package metrics

import "testing"

func TestInitCanBeCalledMultipleTimes(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Init() panicked: %v", r)
		}
	}()

	Init()
	Init()
}
