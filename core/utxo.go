// Copyright (c) 2018 ContentBox Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package core

import (
	"sync"

	corepb "github.com/BOXFoundation/Quicksilver/core/pb"
	"github.com/BOXFoundation/Quicksilver/core/types"
	"github.com/BOXFoundation/Quicksilver/crypto"
	"github.com/BOXFoundation/Quicksilver/storage"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	proto "github.com/gogo/protobuf/proto"
)

// UtxoUnspentCache define unspent transaction  pool
type UtxoUnspentCache struct {
	outPointMap map[types.OutPoint]*UtxoWrap
	tailHash    *types.Block
}

// UtxoWrap utxo wrap
type UtxoWrap struct {
	Value        int64
	ScriptPubKey []byte
	BlockHeight  int
	IsPacked     bool
	IsCoinbase   bool
}

// NewUtxoUnspentCache returns a new empty unspent transaction pool
func NewUtxoUnspentCache() *UtxoUnspentCache {
	return &UtxoUnspentCache{
		outPointMap: make(map[types.OutPoint]*UtxoWrap),
	}
}

// Serialize utxo wrap to proto message.
func (uw *UtxoWrap) Serialize() (proto.Message, error) {
	return &corepb.UtxoWrap{
		Value:        uw.Value,
		ScriptPubKey: uw.ScriptPubKey,
		BlockHeight:  int32(uw.BlockHeight),
		IsPacked:     uw.IsPacked,
		IsCoinbase:   uw.IsCoinbase,
	}, nil
}

// Deserialize convert proto message to utxo wrap.
func (uw *UtxoWrap) Deserialize(message proto.Message) error {

	if message, ok := message.(*corepb.UtxoWrap); ok {
		if message != nil {
			uw.Value = message.Value
			uw.ScriptPubKey = message.ScriptPubKey
			uw.BlockHeight = int(message.BlockHeight)
			uw.IsPacked = message.IsPacked
			uw.IsCoinbase = message.IsCoinbase
			return nil
		}
		return types.ErrEmptyProtoMessage
	}

	return types.ErrEmptyProtoMessage
}

//LoadUtxoFromDB load related unspent utxo
func (uup *UtxoUnspentCache) LoadUtxoFromDB(db storage.Storage, outpoints map[types.OutPoint]struct{}) error {
	return nil
}

// FindByOutPoint returns utxo wrap from cache by outpoint.
func (uup *UtxoUnspentCache) FindByOutPoint(outpoint types.OutPoint) *UtxoWrap {
	return uup.outPointMap[outpoint]
}

// RemoveByOutPoint delete utxo wrap from cache by outpoint.
func (uup *UtxoUnspentCache) RemoveByOutPoint(outpoint types.OutPoint) {
	delete(uup.outPointMap, outpoint)
}

// make sure save unspent utxo to storage when link block to main chain.
func (uup *UtxoUnspentCache) storeUnspentUtxo(db storage.Storage) error {

	for k, v := range uup.outPointMap {
		if v == nil {
			continue
		}
		// Remove the utxo if it is spent.
		if v.IsPacked {
			key := outpointKey(k)
			err := db.Del(*key)
			outpointKeyPool.Put(key)
			if err != nil {
				return err
			}
			continue
		}
		// Serialize and store the utxo.
		serialized, err := v.Serialize()
		if err != nil {
			return err
		}
		serializedBin, err := proto.Marshal(serialized)
		if err != nil {
			return err
		}
		key := outpointKey(k)
		return db.Put(*key, serializedBin)
	}
	return nil

}

func serializeSizeVLQ(n uint64) int {
	size := 1
	for ; n > 0x7f; n = (n >> 7) - 1 {
		size++
	}

	return size
}

func putVLQ(target []byte, n uint64) int {
	offset := 0
	for ; ; offset++ {
		highBitMask := byte(0x80)
		if offset == 0 {
			highBitMask = 0x00
		}

		target[offset] = byte(n&0x7f) | highBitMask
		if n <= 0x7f {
			break
		}
		n = (n >> 7) - 1
	}

	// Reverse the bytes so it is MSB-encoded.
	for i, j := 0, offset; i < j; i, j = i+1, j-1 {
		target[i], target[j] = target[j], target[i]
	}

	return offset + 1
}

var maxUint32VLQSerializeSize = serializeSizeVLQ(1<<32 - 1)

var outpointKeyPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, crypto.HashSize+maxUint32VLQSerializeSize)
		return &b // Pointer to slice to avoid boxing alloc.
	},
}

func outpointKey(outpoint types.OutPoint) *[]byte {
	key := outpointKeyPool.Get().(*[]byte)
	idx := uint64(outpoint.Index)
	*key = (*key)[:chainhash.HashSize+serializeSizeVLQ(idx)]
	copy(*key, outpoint.Hash[:])
	putVLQ((*key)[chainhash.HashSize:], idx)
	return key
}
