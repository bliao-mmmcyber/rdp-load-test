package guac

import "testing"

func TestMemorySessionDataStore(t *testing.T) {
	store := NewMemorySessionDataStore()

	if v, ok := store.Get("a").(int); ok {
		t.Errorf("Expected 'a' to be nil, got %v", v)
	}

	store.Set("a", 123)

	if v := store.Get("a").(int); v != 123 {
		t.Errorf("Expected 'a' to be 123, got %v", v)
	}

	store.Delete("a")

	if v, ok := store.Get("a").(int); ok {
		t.Errorf("Expected 'a' to be nil, got %v", v)
	}

	type Foo struct {
		Value1 int
		Field2 string
		Bool3  bool
	}

	store.Set("foo", Foo{
		Value1: 42,
		Field2: "bar",
	})

	if _, ok := store.Get("foo").(int); ok {
		t.Errorf("Expected 'foo' to be type Foo, got int")
	}

	if _, ok := store.Get("foo").(*Foo); ok {
		t.Errorf("Expected 'foo' to be type Foo, got *Foo")
	}

	if foo, ok := store.Get("foo").(Foo); !ok {
		t.Errorf("Expected 'foo' to be type Foo, but typecast failed")
	} else {
		if foo.Value1 != 42 {
			t.Errorf("Expected 'foo' to have value 42, but got %d", foo.Value1)
		}
	}

	store.Set("and_foo", &Foo{
		Value1: 100,
		Field2: "bar",
		Bool3:  true,
	})

	if foo, ok := store.Get("and_foo").(*Foo); !ok {
		t.Errorf("Expected 'and_foo' to be type *Foo, but typecast failed")
	} else {
		if foo.Value1 != 100 {
			t.Errorf("Expected 'foo' to have value 42, but got %d", foo.Value1)
		}
	}
}
