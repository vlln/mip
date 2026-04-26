package engine

import "testing"

func TestNewSupportedEngines(t *testing.T) {
	for _, name := range Names() {
		got, err := New(name)
		if err != nil {
			t.Fatalf("New(%q) returned error: %v", name, err)
		}
		if got.Name() != name {
			t.Fatalf("New(%q) returned engine %q", name, got.Name())
		}
	}
}

func TestNewDefaultEngine(t *testing.T) {
	got, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "docker" {
		t.Fatalf("default engine = %q, want docker", got.Name())
	}
}

func TestNewUnsupportedEngine(t *testing.T) {
	if _, err := New("missing"); err == nil {
		t.Fatal("expected unsupported engine error")
	}
}
