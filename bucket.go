package cbgb

import (
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

const (
	MAX_VBUCKET        = 1024
	DEFAULT_BUCKET_KEY = "default"
)

type vbucketChange struct {
	bucket             *bucket
	vbid               uint16
	oldState, newState VBState
}

func (c vbucketChange) getVBucket() *vbucket {
	return c.bucket.getVBucket(c.vbid)
}

func (c vbucketChange) String() string {
	return fmt.Sprintf("vbucket %v %v -> %v",
		c.vbid, c.oldState, c.newState)
}

type bucket struct {
	vbuckets    [MAX_VBUCKET]unsafe.Pointer
	availablech chan bool
	observer    *broadcaster
}

func NewBucket() *bucket {
	return &bucket{
		observer:    newBroadcaster(0),
		availablech: make(chan bool),
	}
}

// Holder of buckets
type Buckets struct {
	buckets map[string]*bucket
	lock    sync.Mutex
}

// Build a new holder of buckets.
func NewBuckets() *Buckets {
	return &Buckets{buckets: map[string]*bucket{}}
}

// Create a new named bucket.
// Return the new bucket, or nil if the bucket already exists.
func (b *Buckets) New(name string) *bucket {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.buckets[name] != nil {
		return nil
	}

	rv := NewBucket()
	b.buckets[name] = rv
	return rv
}

// Get the named bucket (or nil if it doesn't exist).
func (b *Buckets) Get(name string) *bucket {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.buckets[name]
}

// Destroy the named bucket.
func (b *Buckets) Destroy(name string) {
	b.lock.Lock()
	defer b.lock.Unlock()

	bucket := b.buckets[name]
	if bucket != nil {
		bucket.Close()
		delete(b.buckets, name)
	}
}

func (b *bucket) Observer() *broadcaster {
	return b.observer
}

// Subscribe to bucket events.
//
// Note that this is retroactive -- it will send existing states.
func (b *bucket) Subscribe(ch chan<- interface{}) {
	b.observer.Register(ch)
	go func() {
		for i := uint16(0); i < MAX_VBUCKET; i++ {
			c := vbucketChange{bucket: b,
				vbid:     i,
				oldState: VBDead,
				newState: VBDead}
			vb := c.getVBucket()
			if vb != nil {
				s := vb.GetVBState()
				if s != VBDead {
					c.newState = s
					ch <- c
				}
			}
		}
	}()
}

func (b *bucket) Close() error {
	close(b.availablech)
	return nil
}

func (b *bucket) Available() bool {
	select {
	default:
	case <-b.availablech:
		return false
	}
	return true
}

func (b *bucket) getVBucket(vbid uint16) *vbucket {
	if b == nil || !b.Available() {
		return nil
	}
	vbp := atomic.LoadPointer(&b.vbuckets[vbid])
	return (*vbucket)(vbp)
}

func (b *bucket) casVBucket(vbid uint16, vb *vbucket, vbPrev *vbucket) bool {
	return atomic.CompareAndSwapPointer(&b.vbuckets[vbid],
		unsafe.Pointer(vbPrev), unsafe.Pointer(vb))
}

func (b *bucket) CreateVBucket(vbid uint16) *vbucket {
	if b == nil || !b.Available() {
		return nil
	}
	vb := newVBucket(b, vbid)
	if b.casVBucket(vbid, vb, nil) {
		return vb
	}
	return nil
}

func (b *bucket) destroyVBucket(vbid uint16) (destroyed bool) {
	destroyed = false
	vb := b.getVBucket(vbid)
	if vb != nil {
		vb.SetVBState(VBDead, func(oldState VBState) {
			if b.casVBucket(vbid, nil, vb) {
				b.observer.Submit(vbucketChange{b, vbid, oldState, VBDead})
				destroyed = true
			}
		})
	}
	return
}

func (b *bucket) SetVBState(vbid uint16, newState VBState) *vbucket {
	vb := b.getVBucket(vbid)
	if vb != nil {
		vb.SetVBState(newState, func(oldState VBState) {
			if b.getVBucket(vbid) == vb {
				b.observer.Submit(vbucketChange{b, vbid, oldState, newState})
			} else {
				vb = nil
			}
		})
	}
	return vb
}
