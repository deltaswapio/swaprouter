package main

import (
	"encoding/json"
	"errors"
	"flag"
	"time"

	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/mpc"
	"github.com/deltaswapio/swaprouter/v3/params"
	"github.com/deltaswapio/swaprouter/v3/tokens"
	"github.com/deltaswapio/swaprouter/v3/tokens/aptos"
)

var (
	bridge = aptos.NewCrossChainBridge()

	paramConfigFile string
	paramChainID    string

	paramPublicKey string
	paramPriKey    string

	coin           string
	poolCoinName   string
	poolCoinSymbol string
	decimals       uint
	monitor_supply bool

	mpcConfig *mpc.Config
)

func initFlags() {
	flag.StringVar(&paramConfigFile, "config", "", "config file to init mpc and gateway")
	flag.StringVar(&paramChainID, "chainID", "", "chain id")

	flag.StringVar(&paramPublicKey, "pubkey", "", "signer public key")
	flag.StringVar(&paramPriKey, "priKey", "", "signer priKey key")

	flag.StringVar(&coin, "coin", "", "coin resource: 0xc441fa1354b4544457df58b7bfdf53fae75e0d6f61ded55b72ae058d2d407c9d::Test02::Coin")

	flag.StringVar(&poolCoinName, "name", "", "anycoin name")
	flag.StringVar(&poolCoinSymbol, "symbol", "", "anycoin symbol")
	flag.UintVar(&decimals, "decimals", 6, "anycoin decimals")
	flag.BoolVar(&monitor_supply, "supply", false, "need monitor supply")

	flag.Parse()
}

func main() {
	log.SetLogger(6, false, true)
	initAll()

	var account *aptos.Account
	if paramPriKey != "" {
		account = aptos.NewAccountFromSeed(paramPriKey)
	} else {
		account = aptos.NewAccountFromPubkey(paramPublicKey)
	}
	log.Info("SignAccount", "address", account.GetHexAddress())
	tx, err := bridge.BuildManagedCoinInitializeTransaction(account.GetHexAddress(), coin, poolCoinName, poolCoinSymbol, uint8(decimals), monitor_supply)
	if err != nil {
		log.Fatalf("%v", err)
	}
	signingMessage, err := bridge.GetSigningMessage(tx)
	if err != nil {
		log.Fatal("GetSigningMessage", "err", err)
	}
	if paramPriKey != "" {
		signature, err := account.SignString(*signingMessage)
		if err != nil {
			log.Fatal("SignString", "err", err)
		}
		tx.Signature = &aptos.TransactionSignature{
			Type:      "ed25519_signature",
			PublicKey: account.GetPublicKeyHex(),
			Signature: signature,
		}
		log.Info("SignTransactionWithPrivateKey", "signature", signature)

	} else {
		mpcPubkey := paramPublicKey

		msgContent := *signingMessage
		jsondata, _ := json.Marshal(tx)
		msgContext := string(jsondata)

		keyID, rsvs, err := mpcConfig.DoSignOneED(mpcPubkey, msgContent, msgContext)
		if err != nil {
			log.Fatal("DoSignOneED", "err", err)
		}
		log.Info("DoSignOneED", "keyID", keyID)

		if len(rsvs) != 1 {
			log.Fatal("DoSignOneED", "err", errors.New("get sign status require one rsv but return many"))
		}
		rsv := rsvs[0]
		tx.Signature = &aptos.TransactionSignature{
			Type:      "ed25519_signature",
			PublicKey: mpcPubkey,
			Signature: rsv,
		}
		log.Info("DoSignOneED", "signature", rsv)
	}
	txInfo, err := bridge.SubmitTranscation(tx)
	if err != nil {
		log.Fatal("SignString", "err", err)
	}
	time.Sleep(time.Duration(10) * time.Second)
	result, _ := bridge.GetTransactions(txInfo.Hash)
	log.Info("SubmitTranscation", "txHash", txInfo.Hash, "Success", result.Success, "version", result.Version, "vm_status", result.VmStatus)
}

func initAll() {
	initFlags()
	initConfig()
	initBridge()
}
func initConfig() {
	config := params.LoadRouterConfig(paramConfigFile, true, false)
	if config.FastMPC != nil {
		mpcConfig = mpc.InitConfig(config.FastMPC, true)
	} else {
		mpcConfig = mpc.InitConfig(config.MPC, true)
	}
	log.Info("init config finished", "IsFastMPC", mpcConfig.IsFastMPC)
}

func initBridge() {
	cfg := params.GetRouterConfig()
	apiAddrs := cfg.Gateways[paramChainID]
	if len(apiAddrs) == 0 {
		log.Fatal("gateway not found for chain ID", "chainID", paramChainID)
	}
	apiAddrsExt := cfg.GatewaysExt[paramChainID]
	bridge.SetGatewayConfig(&tokens.GatewayConfig{
		APIAddress:    apiAddrs,
		APIAddressExt: apiAddrsExt,
	})
	log.Info("init bridge finished")
}
