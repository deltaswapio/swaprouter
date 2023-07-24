package main

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/tokens/btc"
)

var (
	paramWif     string
	paramChainID string
)

func initFlags() {
	flag.StringVar(&paramWif, "wif", "", "wif")
	flag.StringVar(&paramChainID, "chainID", "", "chainID")

	flag.Parse()
}

func main() {
	log.SetLogger(6, false, true)

	initFlags()

	if paramWif == "" {
		log.Fatal("miss network argument")
	}

	wifPd, err := btc.DecodeWIF(paramWif)
	if err != nil {
		log.Fatal("DecodeWIF fails", "paramWif", paramWif)
	}
	ecPrikey := wifPd.PrivKey.ToECDSA()
	priString := hex.EncodeToString(ecPrikey.D.Bytes())
	fmt.Printf("%v: %v:\n", paramWif, priString)
}
