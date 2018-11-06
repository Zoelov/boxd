// Copyright (c) 2018 ContentBox Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package chain

import (
	"sync"

	"github.com/BOXFoundation/boxd/core"
	"github.com/BOXFoundation/boxd/crypto"
	"github.com/BOXFoundation/boxd/storage"
	"github.com/BOXFoundation/boxd/util/bloom"
)

// FilterEntry represents a bloom filter for the block of the given hash
type FilterEntry struct {
	Filter    bloom.Filter
	Height    uint32
	BlockHash crypto.HashType
}

// BloomFilterHolder holds all bloom filters in main chain
type BloomFilterHolder interface {
	ResetFilters(uint32) error
	ListMatchedBlockHashes([]byte) []crypto.HashType
	AddFilter(uint32, crypto.HashType, storage.Table, func() bloom.Filter) error
}

// NewFilterHolder creates an holder instance
func NewFilterHolder() BloomFilterHolder {
	return &MemoryBloomFilterHolder{
		entries: make([]*FilterEntry, 0),
		mux:     &sync.Mutex{},
	}
}

// MemoryBloomFilterHolder holds all bloom filters in main chain in an array format in memory
type MemoryBloomFilterHolder struct {
	entries []*FilterEntry
	mux     *sync.Mutex
}

// AddFilter adds a filter of block at height. Filter is loaded from db instance if it is
// stored, otherwise, it's calculated using onCacheMiss function
func (holder *MemoryBloomFilterHolder) AddFilter(
	height uint32,
	hash crypto.HashType,
	db storage.Table,
	onCacheMiss func() bloom.Filter) error {
	holder.mux.Lock()
	defer holder.mux.Unlock()

	if len(holder.entries) != int(height-1) {
		return core.ErrInvalidFilterHeight
	}
	filterKey := FilterKey(hash)

	// filter stored in db
	if buf, err := db.Get(filterKey); err == nil && buf != nil {
		if filter, err := bloom.LoadFilter(buf); err == nil {
			return holder.addFilterInternal(filter, height, hash)
		}
	}

	// recalculate filter
	filter := onCacheMiss()
	holder.addFilterInternal(filter, height, hash)

	if filterBytes, err := filter.Marshal(); err != nil {
		logger.Error("Error marshal filter for block ", hash.String())
	} else {
		db.Put(filterKey, filterBytes)
	}
	return nil
}

// AddFilter adds a filter of block at height
func (holder *MemoryBloomFilterHolder) addFilterInternal(filte bloom.Filter, height uint32, hash crypto.HashType) error {
	holder.entries = append(holder.entries, &FilterEntry{
		Filter:    filte,
		Height:    height,
		BlockHash: hash,
	})
	return nil
}

// ResetFilters resets filterEntry array to a height
func (holder *MemoryBloomFilterHolder) ResetFilters(height uint32) error {
	holder.mux.Lock()
	defer holder.mux.Unlock()
	if len(holder.entries) < int(height-1) {
		return core.ErrInvalidFilterHeight
	}
	holder.entries = holder.entries[:height-1]
	return nil
}

// ListMatchedBlockHashes search all blocks' bloom filter, and returns block hashes
// that might contain a certain word
func (holder *MemoryBloomFilterHolder) ListMatchedBlockHashes(word []byte) []crypto.HashType {
	holder.mux.Lock()
	defer holder.mux.Unlock()

	matched := make([]crypto.HashType, 0)
	for _, entry := range holder.entries {
		if entry.Filter.Matches(word) {
			matched = append(matched, entry.BlockHash)
		}
	}
	return matched
}
