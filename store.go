package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// file name -> clown.jpg
// path > transformFunc(fileName) -> root/key/path
// key -> can help us to fetch all our files

const defaultRootFolderName = "godfs"

type PathTransformFunc func(string) PathKey

type PathKey struct {
	PathName string
	FileName string
}

func (p PathKey) FirstPathName() string {
	paths := strings.Split(p.PathName, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) FullPathName() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.FileName)
}

type StoreOpts struct {
	Root              string // Root holds the root path
	PathTransformFunc PathTransformFunc
}

var DefaultPathTransportFunc = func(key string) PathKey {
	return PathKey{
		PathName: key,
		FileName: key,
	}
}

func CASPathTransportFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashString := hex.EncodeToString(hash[:])

	blocksize := 5
	sliceLen := len(hashString) / blocksize
	paths := make([]string, sliceLen)
	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashString[from:to]
	}

	return PathKey{
		PathName: strings.Join(paths, "/"),
		FileName: hashString,
	}
}

type Store struct {
	StoreOpts
}

func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransportFunc
	}
	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Has(id, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPathName())

	_, err := os.Stat(fullPathWithRoot)

	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Delete(id, key string) error {
	pathKey := s.PathTransformFunc(key)
	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FirstPathName())

	defer func() {
		log.Printf("deleted [%s] from disk", pathKey.FileName)
	}()

	return os.RemoveAll(firstPathNameWithRoot)
}

func (s *Store) Read(id, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
}

func (s *Store) readStream(id, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPathName())

	file, err := os.Open(fullPathWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fi, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}

	return fi.Size(), file, nil

}

func (s *Store) Write(id, key string, r io.Reader) (int64, error) {
	// getting encrypted data, need to decrypt it before saving
	return s.writeStream(id, key, r)
}

func (s *Store) WriteDecrypt(encKey []byte, id, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	n, err := copyDecrypt(encKey, r, f)
	return int64(n), err
}

func (s *Store) writeStream(id, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	return io.Copy(f, r)
}

func (s *Store) openFileForWriting(id, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.PathName)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPathWithRoot := fmt.Sprintf("%s/%s/%s", s.Root, id, pathKey.FullPathName())

	return os.Create(fullPathWithRoot)

}
