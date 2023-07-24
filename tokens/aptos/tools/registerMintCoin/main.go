package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/deltaswapio/swaprouter/v3/common"
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

	paramPath string

	mpcConfig *mpc.Config
)

func initFlags() {
	flag.StringVar(&paramConfigFile, "config", "", "config file to init mpc and gateway")
	flag.StringVar(&paramChainID, "chainID", "", "chain id")

	flag.StringVar(&paramPublicKey, "pubkey", "", "signer public key")
	flag.StringVar(&paramPriKey, "priKey", "", "signer priKey key")

	flag.StringVar(&paramPath, "path", "", "contract build path: /Users/potti/multichain-workspace/aptos-contract/registerMintCoin/build/RegisterMintCoin")

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

	accountInfo, err := bridge.GetAccount(account.GetHexAddress())
	if err != nil {
		log.Fatal("GetAccount", "err", err)
	}
	moduleHex := readMove(paramPath + "/bytecode_scripts/main.mv")
	// 10 min
	timeout := time.Now().Unix() + 600
	tx := &aptos.ScriptTransaction{
		Sender:                  account.GetHexAddress(),
		SequenceNumber:          accountInfo.SequenceNumber,
		MaxGasAmount:            "20000",
		GasUnitPrice:            "1000",
		ExpirationTimestampSecs: strconv.FormatInt(timeout, 10),
		Payload: &aptos.ScriptPayload{
			Type: aptos.SCRIPT_PAYLOAD,
			Code: aptos.ScriptPayloadCode{
				Bytecode: moduleHex,
			},
			// TypeArguments: []string{account.GetHexAddress() + "::DAI::Coin", account.GetHexAddress() + "::ETH::Coin", account.GetHexAddress() + "::USDC::Coin", account.GetHexAddress() + "::USDT::Coin", account.GetHexAddress() + "::WBTC::Coin"},
			TypeArguments: []string{},
			Arguments:     []interface{}{},
		},
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
		log.Fatal("SubmitTranscation", "err", err)
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

func readMove(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("readMove", "filename", filename)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("ReadAll", "filename", filename)
	}
	return common.ToHex(content)
}
