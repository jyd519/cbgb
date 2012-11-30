package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"testing"

	"github.com/dustin/gomemcached"
)

// Don't do any normal logging while running tests.
func init() {
	log.SetOutput(ioutil.Discard)
}

// Exercise the mutation logger code. Output is not examined.
func TestMutationLogger(t *testing.T) {
	ch := make(chan interface{}, 5)
	ch <- mutation{deleted: false, key: []byte("a"), cas: 0}
	ch <- mutation{deleted: true, key: []byte("a"), cas: 0}
	ch <- mutation{deleted: false, key: []byte("a"), cas: 2}
	ch <- bucketChange{deleted: false, vbid: 0}
	ch <- bucketChange{deleted: true, vbid: 0}
	close(ch)

	mutationLogger(ch)
}

func TestMutationInvalid(t *testing.T) {
	ch := make(chan interface{}, 5)
	ch <- 19

	func() {
		defer func() {
			if x := recover(); x == nil {
				t.Fatalf("Expected panic, didn't get it")
			} else {
				t.Logf("Got expected panic in invalid mutation: %v", x)
			}
		}()
		mutationLogger(ch)
	}()
}

// Run through the sessionLoop code with a quit command.
//
// This test doesn't do much other than confirm that the session loop
// actually would terminate the real session goroutine on quit (by
// completing).
func TestSessionLoop(t *testing.T) {
	req := &gomemcached.MCRequest{
		Opcode: gomemcached.QUIT,
	}

	rh := &reqHandler{}

	req.Bytes()
	sessionLoop(rwCloser{bytes.NewBuffer(req.Bytes())}, "test", rh)
}
