package cbgb

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/dustin/gomemcached"
	"github.com/robertkrimen/otto"
)

type DDocs map[string]*DDoc

type DDoc struct {
	Language string       `json:"language,omitempty"`
	Views    Views        `json:"views,omitempty"`
	Options  *DDocOptions `json:"options,omitempty"`
}

type DDocOptions struct {
	LocalSeq      bool `json:"local_seq,omitempty"`
	IncludeDesign bool `json:"include_design,omitempty"`
}

type Views map[string]*View

type View struct {
	Map    string `json:"map"`
	Reduce string `json:"reduce,omitempty"`
}

func (b *livebucket) GetDDocVBucket() *VBucket {
	return b.vbucketDDoc
}

func (b *livebucket) GetDDoc(ddocId string) ([]byte, error) {
	res := b.vbucketDDoc.get([]byte(ddocId))
	if res.Status == gomemcached.KEY_ENOENT {
		return nil, nil
	}
	if res.Status != gomemcached.SUCCESS {
		return nil, fmt.Errorf("no ddoc: %v, status: %v", ddocId, res.Status)
	}
	return res.Body, nil
}

func (b *livebucket) SetDDoc(ddocId string, body []byte) error {
	res := vbMutate(b.vbucketDDoc, nil, &gomemcached.MCRequest{
		Opcode: gomemcached.SET,
		Key:    []byte(ddocId),
		Body:   body,
	})
	if res.Status != gomemcached.SUCCESS {
		return fmt.Errorf("set ddoc failed: %v, status: %v", ddocId, res.Status)
	}
	return nil
}

func (b *livebucket) VisitDDocs(start []byte,
	visitor func(key []byte, data []byte) bool) error {
	return b.vbucketDDoc.Visit(start, visitor)
}

func (b *livebucket) GetDDocs() *DDocs {
	ddocs := (*DDocs)(atomic.LoadPointer(&b.ddocs))
	if ddocs == nil {
		for {
			ddocs = &DDocs{}
			var errJson error
			var errVisit error
			errVisit = b.VisitDDocs(nil, func(key []byte, data []byte) bool {
				ddoc := &DDoc{}
				errJson = json.Unmarshal(data, ddoc)
				if errJson != nil {
					// TODO: Perhaps should continue on with rest of ddocs
					// if any ddoc fails to parse; and log the error somewhere.
					return false
				}
				(*ddocs)[string(key)] = ddoc
				return true
			})
			if errVisit != nil || errJson != nil {
				return nil
			}
			if b.SetDDocs(nil, ddocs) {
				break
			}
		}
	}
	return ddocs
}

func (b *livebucket) SetDDocs(old, val *DDocs) bool {
	return atomic.CompareAndSwapPointer(&b.ddocs,
		unsafe.Pointer(old), unsafe.Pointer(val))
}

func (v *View) GetMapFunction() (otto.Value, error) {
	return otto.UndefinedValue(), nil
}
