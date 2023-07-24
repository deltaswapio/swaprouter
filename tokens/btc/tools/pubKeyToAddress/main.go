package main

import (
	"flag"
	"fmt"

	"github.com/btcsuite/btcutil"
	"github.com/deltaswapio/swaprouter/v3/common"
	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/tokens/btc"
)

var (
	paramPubKey  string
	paramChainID string
)

func initFlags() {
	flag.StringVar(&paramPubKey, "pubKey", "", "pubKey")
	flag.StringVar(&paramChainID, "chainID", "", "chainId")

	flag.Parse()
}

func main() {
	log.SetLogger(6, false, true)

	initFlags()

	if paramPubKey == "" {
		log.Fatal("miss paramPubKey argument")
	}

	pkData := common.FromHex(paramPubKey)
	b := btc.NewCrossChainBridge()
	cPkData, err := b.ToCompressedPublicKey(pkData)
	if err != nil {
		log.Fatal("ToCompressedPublicKey fails", "paramPubKey", paramPubKey)
	}
	chainID, err := common.GetBigIntFromStr(paramChainID)
	if err != nil {
		log.Fatal("GetBigIntFromStr fails", "paramChainID", paramChainID)
	}
	chainParams := b.GetChainParams(chainID)
	address, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(cPkData), chainParams)
	if err != nil {
		log.Fatal("NewAddressPubKeyHash fails", "paramPubKey", paramPubKey)
	}
	fmt.Printf("address: %v\n", address)
}
