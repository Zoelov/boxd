// Copyright (c) 2018 ContentBox Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package p2p

import (
	conv "github.com/BOXFoundation/boxd/p2p/convert"
	peer "github.com/libp2p/go-libp2p-peer"
)

// DummyPeer implements Net interface for testing purpose
type DummyPeer struct{}

// NewDummyPeer creates a new DummyPeer
func NewDummyPeer() *DummyPeer {
	return &DummyPeer{}
}

// Broadcast for testing
func (d *DummyPeer) Broadcast(uint32, conv.Convertible) error {
	return nil
}

// SendMessageToPeer for testing
func (d *DummyPeer) SendMessageToPeer(uint32, conv.Convertible, peer.ID) {}

// Subscribe for testing
func (d *DummyPeer) Subscribe(*Notifiee) {}

// UnSubscribe for testing
func (d *DummyPeer) UnSubscribe(*Notifiee) {}

// Notify for testing
func (d *DummyPeer) Notify(Message) {}
