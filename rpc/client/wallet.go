// Copyright (c) 2018 ContentBox Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package client

import (
	"context"
	"log"
	"time"

	"github.com/BOXFoundation/boxd/core/types"
	"github.com/BOXFoundation/boxd/rpc/pb"
	"github.com/spf13/viper"
)

// ListTransactions list transactions of certain address
func ListTransactions(v *viper.Viper, addr string, offset, limit uint32) ([]*types.Transaction, error) {
	conn := mustConnect(v)
	defer conn.Close()
	c := rpcpb.NewWalletCommandClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Printf("List Transactions of address: %s", addr)

	r, err := c.ListTransactions(ctx, &rpcpb.ListTransactionsRequest{Addr: addr, Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}

	txs := make([]*types.Transaction, len(r.Transactions))
	for i, rpcTx := range r.Transactions {
		tx := &types.Transaction{}
		err = tx.FromProtoMessage(rpcTx)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	return txs, nil
}
