// Copyright (c) 2018 ContentBox Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package splitaddrcmd

import (
	"encoding/hex"
	"errors"
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
			Use:   "create [(addr1, weight1), (addr2, weight2), (addr3, weight3), ...]",
			Short: "Create a split address from multiple addresses and their weights: address order matters",
			Run:   createCmdFunc,
		},
		&cobra.Command{
			Use:   "sendfrom fromaddr toSplitAddr amount",
			Short: "send from regular address to a split address",
			Run:   sendFromCmdFunc,
		},
		&cobra.Command{
			Use:   "redeem addr2 toAddr amount [(addr1, weight1), (addr2, weight2), (addr3, weight3), ...]",
			Short: "send from split address to regular address",
			Run:   redeemCmdFunc,
		},
		&cobra.Command{
			Use:   "getbalance [(addr1, weight1), (addr2, weight2), (addr3, weight3), ...]",
			Short: "Get balances for all addresses in a split address",
			Run:   getBalanceCmdFunc,
		},
	)
}

func createCmdFunc(cmd *cobra.Command, args []string) {
	fmt.Println("create called")
	if len(args) < 2 || len(args)%2 == 1 {
		fmt.Println("Invalid argument number: expect even number")
		return
	}
	pubKeys, weights, err := parsePubKeyWeight(args)
	if err != nil {
		fmt.Println(err)
		return
	}
	splitAddr, _, err := createSplitAddr(pubKeys, weights)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Split address generated for `%s`: %s\n", args, splitAddr)
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
	tx, err := client.CreateTransaction(conn, fromAddr, fromAddr, target, false, /* from addr not split */
		true /* to addr split */, account.PublicKey(), script.InvalidIdx, account)
	if err != nil {
		fmt.Println(err)
	} else {
		hash, _ := tx.TxHash()
		fmt.Println("Tx Hash:", hash.String())
		fmt.Println(util.PrettyPrint(tx))
	}
}

func redeemCmdFunc(cmd *cobra.Command, args []string) {
	// pubkey index starts from 1
	// redeem pubkey_idx2 toAddr amount [(addr1, weight1), (addr2, weight2), (addr3, weight3), ...]
	if len(args) < 5 {
		fmt.Println("Invalid argument number")
		return
	}
	target, err := root.ParseSendTarget(args[1:3])
	if err != nil {
		fmt.Println(err)
		return
	}
	pubKeys, weights, err := parsePubKeyWeight(args[3:])
	if err != nil {
		fmt.Println(err)
		return
	}
	splitAddr, splitScript, err := createSplitAddr(pubKeys, weights)
	if err != nil {
		fmt.Println(err)
		return
	}
	wltMgr, err := wallet.NewWalletManager(walletDir)
	if err != nil {
		fmt.Println(err)
		return
	}

	pubKeyIdx, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	// pubKeyIdx starts from 1
	pubKeyBytes := pubKeys[pubKeyIdx-1]
	pubKey, err := crypto.PublicKeyFromBytes(pubKeyBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	pubKeyHash, err := types.NewAddressFromPubKey(pubKey)
	if err != nil {
		fmt.Println(err)
		return
	}
	addr := pubKeyHash.String()
	account, exists := wltMgr.GetAccount(addr)
	if !exists {
		fmt.Printf("Account %s not managed\n", addr)
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
	fromSplitAddr, err := types.NewAddress(splitAddr)
	if err != nil {
		fmt.Println("Invalid address: ", splitAddr)
	}
	changeAddr, err := types.NewAddress(addr)
	if err != nil {
		fmt.Println("Invalid address: ", addr)
	}
	conn := client.NewConnectionWithViper(viper.GetViper())
	defer conn.Close()

	// p2pkh unlock: sig + pubKey
	// split unlock: sig + sig index + redeem script
	tx, err := client.CreateTransaction(conn, fromSplitAddr, changeAddr, target, true, /* from addr split */
		false /* to addr not split */, *splitScript /* at the same place as public key*/, pubKeyIdx, account)
	if err != nil {
		fmt.Println(err)
	} else {
		hash, _ := tx.TxHash()
		fmt.Println("Tx Hash:", hash.String())
		fmt.Println(util.PrettyPrint(tx))
	}
}

func getBalanceCmdFunc(cmd *cobra.Command, args []string) {
	fmt.Println("getBalance called")
	if len(args) < 2 || len(args)%2 == 1 {
		fmt.Println("Invalid argument number: expect even number")
		return
	}
	pubKeys, weights, err := parsePubKeyWeight(args)
	if err != nil {
		fmt.Println(err)
		return
	}
	splitAddr, _, err := createSplitAddr(pubKeys, weights)
	if err != nil {
		fmt.Println(err)
		return
	}

	conn := client.NewConnectionWithViper(viper.GetViper())
	defer conn.Close()
	balances, err := client.GetBalance(conn, []string{splitAddr}, true /* split address */)
	if err != nil {
		fmt.Println(err)
		return
	}
	totalBalance := balances[splitAddr]
	var totalWeight uint64
	for _, weight := range weights {
		totalWeight += weight
	}
	for i, addr := range pubKeys {
		fmt.Printf("Address: %v\t balance: %d\n", addr, totalBalance*weights[i]/totalWeight)
	}
	fmt.Println("Total balance: ", totalBalance)
}

func parsePubKeyWeight(args []string) ([][]byte, []uint64, error) {
	pubKeys := make([][]byte, 0)
	weights := make([]uint64, 0)
	for i := 0; i < len(args)/2; i++ {
		pubKey, err := hex.DecodeString(args[i*2])
		if err != nil {
			return nil, nil, err
		}
		pubKeys = append(pubKeys, pubKey)

		weight, err := strconv.Atoi(args[i*2+1])
		if err != nil {
			return nil, nil, err
		}
		weights = append(weights, uint64(weight))
	}
	return pubKeys, weights, nil
}

// create a split address from arguments
func createSplitAddr(pubKeys [][]byte, weights []uint64) (string, *script.Script, error) {
	s := script.SplitAddrScript(pubKeys, weights)
	if s == nil {
		return "", nil, errors.New("Generate split address error")
	}

	scriptHash := crypto.Hash160(*s)
	splitAddr, err := types.NewAddressPubKeyHash(scriptHash)
	return splitAddr.String(), s, err
}
