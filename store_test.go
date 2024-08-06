package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

// func TestPathTransPortFunc(t *testing.T) {
// 	key := "mombestpicture"
// 	pathname := CASPathTransportFunc(key)
// 	fmt.Println(pathname)
// }

func TestStoreDeleteKey(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransportFunc,
	}
	s := NewStore(opts)
	id := generateId()
	key := "momsspecials"
	data := []byte("some jpeg bytes")
	if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := s.Delete(id, key); err != nil {
		t.Error(err)
	}

}

func TestStore(t *testing.T) {
	s := newStore()
	id := generateId()
	defer teardown(t, s)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("momsspecials%d", i)
		data := []byte("some jpeg bytes")

		if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); !ok {
			t.Errorf("expected to have key %s", key)
		}

		_, r, err := s.Read(id, key)
		if err != nil {
			t.Error(err)
		}

		b, _ := io.ReadAll(r)
		fmt.Println(string(b))
		if string(b) != string(data) {
			t.Errorf("want %s, have %s", string(data), string(b))
		}

		if err = s.Delete(id, key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(id, key); ok {
			t.Errorf("expected to not have key %s", key)
		}
	}

}

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransportFunc,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}
