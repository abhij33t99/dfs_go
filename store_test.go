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
	key := "momsspecials"
	data := []byte("some jpeg bytes")
	if _,err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}

}

func TestStore(t *testing.T) {
	s := newStore()

	defer teardown(t, s)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("momsspecials%d", i)
		data := []byte("some jpeg bytes")

		if _,err := s.writeStream(key, bytes.NewReader(data)); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); !ok {
			t.Errorf("expected to have key %s", key)
		}

		r, err := s.Read(key)
		if err != nil {
			t.Error(err)
		}

		b, _ := io.ReadAll(r)
		fmt.Println(string(b))
		if string(b) != string(data) {
			t.Errorf("want %s, have %s", string(data), string(b))
		}

		if err = s.Delete(key); err != nil {
			t.Error(err)
		}

		if ok := s.Has(key); ok {
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
