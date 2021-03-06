// Copyright 2017 Pilosa Corp.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
// 1. Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright
// notice, this list of conditions and the following disclaimer in the
// documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its
// contributors may be used to endorse or promote products derived
// from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND
// CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES,
// INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR
// CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING,
// BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
// WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING
// NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH
// DAMAGE.

package pdk

import (
	"sync"

	"github.com/pkg/errors"
)

// PeekingSource is a wrapper for Source which implements the
// Peeker interface by reading the next record from Source
// and caching it for the next call to Record().
type PeekingSource struct {
	Source

	mu  sync.RWMutex
	rec interface{}
}

// Peek returns a copy of the next record in the underlying source, without
// discarding it, so the following call to Record() will return the same data
// as if Peek had not been called.
func (p *PeekingSource) Peek() (interface{}, error) {
	p.mu.RLock()
	var err error

	if p.rec != nil {
		defer p.mu.RUnlock()
		return p.rec, nil
	} else {
		// Exchange read lock for write lock and recheck p.rec
		p.mu.RUnlock()
		p.mu.Lock()
		if p.rec != nil {
			p.mu.Unlock()
			return p.Peek()
		}
		p.rec, err = p.Source.Record()
		defer p.mu.Unlock()
		if err != nil {
			return nil, errors.Wrap(err, "getting next record for peeking")
		} else {
			return p.rec, nil
		}
	}
}

// Record returns the next record in the underlying source, first checking if a
// cached record from Peek() has been set.
func (p *PeekingSource) Record() (interface{}, error) {
	p.mu.RLock()
	var rec interface{}
	if p.rec == nil {
		defer p.mu.RUnlock()
		return p.Source.Record()
	} else {
		// Exchange read lock for write lock and recheck p.rec
		p.mu.RUnlock()
		p.mu.Lock()
		if p.rec == nil {
			p.mu.Unlock()
			return p.Record()
		}
		rec = p.rec
		p.rec = nil
		p.mu.Unlock()
		return rec, nil
	}
}

// NewPeekingSource returns a new peeking source.
func NewPeekingSource(source Source) *PeekingSource {
	return &PeekingSource{Source: source}
}
