// Copyright (c) 2018 ContentBox Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package p2p

import (
	"bufio"
	"errors"
	"hash/crc32"

	libp2pnet "github.com/libp2p/go-libp2p-net"
)

// error defined
var (
	ErrMagic               = errors.New("magic is error")
	ErrHeaderCheckSum      = errors.New("header checksum is error")
	ErrExceedMaxDataLength = errors.New("exceed max data length")
	ErrBodyCheckSum        = errors.New("body checksum is error")
)

// Conn represents a connection to a remote node
type Conn struct {
	stream libp2pnet.Stream
	peer   *BoxPeer
}

// NewConn create a stream to remote peer.
func NewConn(stream libp2pnet.Stream, peer *BoxPeer) *Conn {
	return &Conn{stream: stream, peer: peer}
}

func (conn *Conn) readData(rw *bufio.ReadWriter) {

	buf := make([]byte, 1024)
	messageBuffer := make([]byte, 0)
	var header *MessageHeader
	for {
		n, err := rw.Read(buf)
		if err != nil {
			conn.Close()
			return
		}
		messageBuffer = append(messageBuffer, buf[:n]...)
		for {
			if header == nil {
				var err error
				if len(messageBuffer) < MessageHeaderLength {
					break
				}
				headerBytes := messageBuffer[:MessageHeaderLength]
				header, err = ParseHeader(headerBytes)
				if err != nil {
					conn.Close()
					return
				}
				if err := conn.checkHeader(header, headerBytes); err != nil {
					logger.Error("Invalid message header. ", err)
					conn.Close()
					return
				}
				messageBuffer = messageBuffer[MessageHeaderLength:]
			}
			if len(messageBuffer) < int(header.DataLength) {
				break
			}
			body := messageBuffer[:header.DataLength]
			if err := conn.checkBody(header, body); err != nil {
				logger.Error("Invalid message body. ", err)
				conn.Close()
				return
			}
			messageBuffer = messageBuffer[header.DataLength:]
			conn.handle(header.Code, body)
			header = nil
		}
	}

}

func (conn *Conn) writeData(rw *bufio.ReadWriter) {

	for {
		select {}
	}

}

func (conn *Conn) handle(messageCode uint32, body []byte) {

	switch messageCode {
	case Ping:
		conn.onPing(body)
	case Pong:
		conn.onPong(body)
	}
}

func (conn *Conn) Ping() error {
	return nil
}

func (conn *Conn) onPing(data []byte) {

}

func (conn *Conn) onPong(data []byte) {

}

func (conn *Conn) Write(data []byte) error {

	return nil
}

// Close connection to remote peer.
func (conn *Conn) Close() {
	delete(conn.peer.conns, conn.stream.Conn().RemotePeer().String())
	conn.stream.Close()
}

func (conn *Conn) checkHeader(header *MessageHeader, headerBytes []byte) error {
	if conn.peer.config.Magic != header.Magic {
		return ErrMagic
	}
	expectedCheckSum := crc32.ChecksumIEEE(headerBytes)
	if expectedCheckSum != header.Checksum {
		return ErrHeaderCheckSum
	}
	if header.DataLength > MaxNebMessageDataLength {
		return ErrExceedMaxDataLength
	}
	return nil
}

func (conn *Conn) checkBody(header *MessageHeader, body []byte) error {
	expectedDataCheckSum := crc32.ChecksumIEEE(body)
	if expectedDataCheckSum != header.DataChecksum {
		return ErrBodyCheckSum
	}
	return nil
}