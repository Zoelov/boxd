// Copyright (c) 2018 ContentBox Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package root

import (
	"strconv"

	"github.com/BOXFoundation/boxd/core/types"
)

// ParseSendTarget parses command line arguments
func ParseSendTarget(args []string) (map[types.Address]uint64, error) {
	targets := make(map[types.Address]uint64)
	for i := 0; i < len(args)/2; i++ {
		addr, err := types.NewAddress(args[i*2])
		if err != nil {
			return targets, err
		}
		amount, err := strconv.Atoi(args[i*2+1])
		if err != nil {
			return targets, err
		}
		targets[addr] = uint64(amount)
	}
	return targets, nil
}
