package limiter

import "testing"

func TestManagerSharesAggregateBucket(t *testing.T) {
	m := NewManager()
	a := m.ForUser("Alice@example.com", 100, 50)
	b := m.ForUser("alice@example.com", 100, 50)
	if a.Up == nil || a.Down == nil {
		t.Fatal("expected both rate limiters")
	}
	if a.Up != b.Up || a.Down != b.Down {
		t.Fatal("same client/rate must share token buckets across connections")
	}
}

func TestManagerZeroIsUnlimited(t *testing.T) {
	p := NewManager().ForUser("alice", 0, 0)
	if p.Up != nil || p.Down != nil {
		t.Fatal("zero must mean unlimited")
	}
}
