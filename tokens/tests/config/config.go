package config

import (
	"encoding/json"
	"math/big"

	"github.com/BurntSushi/toml"
	"github.com/deltaswapio/swaprouter/v3/common"
	"github.com/deltaswapio/swaprouter/v3/log"
	"github.com/deltaswapio/swaprouter/v3/tokens"
)

var (
	// TestConfig test config instance
	TestConfig = &Config{}

	// ChanIn channel to receive input arguments
	ChanIn = make(chan map[string]string)
	// ChanOut channel to send output result
	ChanOut = make(chan string)
)

// Config test config struct
type Config struct {
	// router swap identifier
	Identifier string

	// router swap type
	SwapType string

	// test module name
	Module string

	// rpc listen port
	Port int

	// sign with this private key instead of MPC
	SignWithPrivateKey string
	SignerAddress      string

	// allow call into router from contract
	AllowCallByContract bool

	// is debug mode (print more logs)
	IsDebugMode bool

	// gatesway config
	Gateway *tokens.GatewayConfig

	// chain config
	Chain *tokens.ChainConfig

	// token config
	Token *tokens.TokenConfig

	// swap config
	Swap *SwapConfig

	// all chain ids (pass 'miss token config' checking)
	AllChainIDs []string
	// calc and cached values
	allChainIDs []*big.Int
}

// GetAllChainIDs get all chainIDs
func (c *Config) GetAllChainIDs() []*big.Int {
	return c.allChainIDs
}

// GetSwapConfig get swap config
func (c *Config) GetSwapConfig() *tokens.SwapConfig {
	return c.Swap.swapConfig
}

// GetFeeConfig get fee config
func (c *Config) GetFeeConfig() *tokens.FeeConfig {
	return c.Swap.feeConfig
}

// SwapConfig swap config
type SwapConfig struct {
	SwapFeeRatePerMillion uint64
	MaximumSwapFee        string
	MinimumSwapFee        string
	MaximumSwap           string
	BigValueThreshold     string
	MinimumSwap           string

	// calc and cached values
	swapConfig *tokens.SwapConfig
	feeConfig  *tokens.FeeConfig
}

// CheckConfig check swap config
func (c *SwapConfig) CheckConfig() error {
	feeConfig := &tokens.FeeConfig{}
	feeConfig.SwapFeeRatePerMillion = c.SwapFeeRatePerMillion
	feeConfig.MaximumSwapFee, _ = common.GetBigIntFromStr(c.MaximumSwapFee)
	feeConfig.MinimumSwapFee, _ = common.GetBigIntFromStr(c.MinimumSwapFee)
	if err := feeConfig.CheckConfig(); err != nil {
		return err
	}

	swapConfig := &tokens.SwapConfig{}
	swapConfig.MaximumSwap, _ = common.GetBigIntFromStr(c.MaximumSwap)
	swapConfig.BigValueThreshold, _ = common.GetBigIntFromStr(c.BigValueThreshold)
	swapConfig.MinimumSwap, _ = common.GetBigIntFromStr(c.MinimumSwap)
	if err := swapConfig.CheckConfig(); err != nil {
		return err
	}

	c.swapConfig = swapConfig
	c.feeConfig = feeConfig
	return nil
}

// LoadTestConfig load test router config
func LoadTestConfig(configFile string) {
	if configFile == "" {
		log.Fatal("must specify config file")
	}
	log.Info("load test config file", "configFile", configFile)
	if !common.FileExist(configFile) {
		log.Fatalf("LoadTestConfig error: config file '%v' not exist", configFile)
	}
	config := &Config{}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatalf("LoadTestConfig error (toml DecodeFile): %v", err)
	}

	var bs []byte
	if log.JSONFormat {
		bs, _ = json.Marshal(config)
	} else {
		bs, _ = json.MarshalIndent(config, "", "  ")
	}
	log.Println("LoadTestConfig finished.", string(bs))

	TestConfig = config

	tokens.InitRouterSwapType(TestConfig.SwapType)

	checkConfig()
}

func checkConfig() {
	if TestConfig.Gateway == nil {
		log.Fatal("must have gateway config")
	}
	if TestConfig.Chain == nil {
		log.Fatal("must have chain config")
	}
	if TestConfig.Token == nil {
		log.Fatal("must have token config")
	}
	if TestConfig.Swap == nil {
		log.Fatal("must have swap config")
	}

	var err error

	if err = TestConfig.Chain.CheckConfig(); err != nil {
		log.Fatal("check chain config failed", "err", err)
	}

	if err = TestConfig.Token.CheckConfig(); err != nil {
		log.Fatal("check token config failed", "err", err)
	}

	if err = TestConfig.Swap.CheckConfig(); err != nil {
		log.Fatal("check swap config failed", "err", err)
	}

	allChainIDs := make([]*big.Int, 0, len(TestConfig.AllChainIDs))
	for _, chainIDStr := range TestConfig.AllChainIDs {
		chainID, err := common.GetBigIntFromStr(chainIDStr)
		if err != nil {
			log.Fatal("wrong chainID in 'AllChainIDs'", "chainID", chainIDStr, "err", err)
		}
		allChainIDs = append(allChainIDs, chainID)
	}
	TestConfig.allChainIDs = allChainIDs
}
