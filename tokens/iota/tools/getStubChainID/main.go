package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/tokens/iota"
)

var (
	paramNetwork string
)

func initFlags() {
	flag.StringVar(&paramNetwork, "p", "", "network, eg. mainnet, testnet, devnet, etc.")

	flag.Parse()
}

func main() {
	log.SetLogger(6, false, true)

	initFlags()

	network := paramNetwork
	if network == "" && len(os.Args) > 1 {
		network = os.Args[1]
	}
	if network == "" {
		log.Fatal("miss network argument")
	}

	chainID := iota.GetStubChainID(network)
	fmt.Printf("%v: %v\n", network, chainID)
}
