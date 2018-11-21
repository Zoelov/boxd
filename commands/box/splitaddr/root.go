// Copyright (c) 2018 ContentBox Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package splitaddrcmd

import (
	"fmt"
	"path"
	"strconv"

	root "github.com/BOXFoundation/boxd/commands/box/root"
	"github.com/BOXFoundation/boxd/core/types"
	"github.com/BOXFoundation/boxd/crypto"
	"github.com/BOXFoundation/boxd/rpc/client"
	"github.com/BOXFoundation/boxd/script"
	"github.com/BOXFoundation/boxd/util"
	"github.com/BOXFoundation/boxd/wallet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var walletDir string
var defaultWalletDir = path.Join(util.HomeDir(), ".box_keystore")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "splitaddr",
	Short: "Split address subcommand",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Init adds the sub command to the root command.
func init() {
	root.RootCmd.AddCommand(rootCmd)
	rootCmd.PersistentFlags().StringVar(&walletDir, "wallet_dir", defaultWalletDir, "Specify directory to search keystore files")
	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "create [(address1, weight1), (addr2, weight2), (addr3, weight3), ...]",
			Short: "Create a split address from multiple addresses and their weights",
			Run:   createCmdFunc,
		},
		&cobra.Command{
			Use:   "sendfrom fromaddr toSplitAddr amount",
			Short: "send to a split address",
			Run:   sendFromCmdFunc,
		},
	)
}

func parseAddrWeight(args []string) ([]types.Address, []int64, error) {
	addrs := make([]types.Address, 0)
	weights := make([]int64, 0)
	for i := 0; i < len(args)/2; i++ {
		addr, err := types.NewAddress(args[i*2])
		if err != nil {
			return nil, nil, err
		}
		addrs = append(addrs, addr)

		weight, err := strconv.Atoi(args[i*2+1])
		if err != nil {
			return nil, nil, err
		}
		weights = append(weights, int64(weight))
	}
	return addrs, weights, nil
}

func createCmdFunc(cmd *cobra.Command, args []string) {
	if len(args) < 2 || len(args)%2 == 1 {
		fmt.Println("Invalid argument number: expect even number")
		return
	}

	addrs, weights, err := parseAddrWeight(args)
	if err != nil {
		fmt.Println(err)
		return
	}

	s := script.SplitAddrScript(addrs, weights)
	if s == nil {
		fmt.Println("Generate split address error")
		return
	}

	scriptHash := crypto.Hash160(*s)
	splitAddr, err := types.NewAddressPubKeyHash(scriptHash)
	if s == nil {
		fmt.Println("Generate split address error")
		return
	}

	fmt.Printf("Split address generated for `%s`: %s\n", args, splitAddr.String())
}

func sendFromCmdFunc(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		fmt.Println("Invalid argument number")
		return
	}
	target, err := root.ParseSendTarget(args[1:])
	if err != nil {
		fmt.Println(err)
		return
	}
	wltMgr, err := wallet.NewWalletManager(walletDir)
	if err != nil {
		fmt.Println(err)
		return
	}
	account, exists := wltMgr.GetAccount(args[0])
	if !exists {
		fmt.Printf("Account %s not managed\n", args[0])
		return
	}
	passphrase, err := wallet.ReadPassphraseStdin()
	if err != nil {
		fmt.Println(err)
		return
	}
	if err := account.UnlockWithPassphrase(passphrase); err != nil {
		fmt.Println("Fail to unlock account", err)
		return
	}
	fromAddr, err := types.NewAddress(args[0])
	if err != nil {
		fmt.Println("Invalid address: ", args[0])
	}
	conn := client.NewConnectionWithViper(viper.GetViper())
	defer conn.Close()
	tx, err := client.CreateTransaction(conn, fromAddr, target, true, account.PublicKey(), account)
	if err != nil {
		fmt.Println(err)
	} else {
		hash, _ := tx.TxHash()
		fmt.Println("Tx Hash:", hash.String())
		fmt.Println(util.PrettyPrint(tx))
	}
}
