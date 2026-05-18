package storage

import (
	"io"
	"os"
    "path/filepath"
)

/*
Some packages that we should use/investigate from the Go standard library:

os — library for interacting with files
some of these are thread safe, some aren't. Need to check documentation
- os.Open / os.Create for reading and writing files
- os.CreateTemp — for creating temp files for atomic writes
- os.Rename
- os.MkdirAll
- os.Remove

io — for streaming.
- io.Copy moves bytes between a reader and a writer without loading everything into memory
- io.Reader - go interface for reading a stream of data
- io.ReadCloser - an interface to close a stream of data

path/filepath — for path manipulation, should be used to convert filepaths into
paths that an os can understand

io/fs — can be used for file path validation and support relative path names
*/

// TODO: Update function signatures and Store struct. This current version is just a draft
type Store struct {
	directory string
}

func New(directory string) (*Store, error) {}

func (s *Store) Put(name string, r io.Reader) error {}

func (s *Store) Get(name string) (io.ReadCloser, error) {}

func (s *Store) List() ([]string, error) {}