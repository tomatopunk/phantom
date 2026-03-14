package session

import (
	"context"
	"testing"
)

func TestManagerGetOrCreateAndClose(t *testing.T) {
	mgr := NewManager("")
	ctx := context.Background()
	s1, err := mgr.GetOrCreate(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if s1.ID != "s1" {
		t.Errorf("session id want s1 got %s", s1.ID)
	}
	s2, err := mgr.GetOrCreate(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if s2 != s1 {
		t.Error("same id should return same session")
	}
	list := mgr.List()
	if len(list) != 1 || list[0] != "s1" {
		t.Errorf("list want [s1] got %v", list)
	}
	mgr.Close("s1")
	if mgr.Get("s1") != nil {
		t.Error("after close Get should return nil")
	}
	if len(mgr.List()) != 0 {
		t.Errorf("list after close want [] got %v", mgr.List())
	}
}
