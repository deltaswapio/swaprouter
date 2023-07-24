package router

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"

	"github.com/deltaswapio/swaprouter/v3/tokens/solana/programs/system"
	"github.com/deltaswapio/swaprouter/v3/tokens/solana/types"
	bin "github.com/streamingfast/binary"
)

// typeID constants
var (
	InitializeTypeID            = calcSighash("global:initialize")              // 0xafaf6d1f0d989bed
	CreateAssociatedTokenTypeID = calcSighash("global:create_associated_token") // 0x9105c275d5740bde
	ChangeMpcTypeID             = calcSighash("global:change_mpc")              // 0x2ba8f0e21522a8ab
	SwapinMintTypeID            = calcSighash("global:swapin_mint")             // 0xbfe596d89e2bfab4
	SwapinTransferTypeID        = calcSighash("global:swapin_transfer")         // 0xc8abfa6f944bb0c4
	SwapinNativeTypeID          = calcSighash("global:swapin_native")           // 0x475cf26f2e26f77a
	SwapoutBurnTypeID           = calcSighash("global:swapout_burn")            // 0x76f70b25faacecef
	SwapoutTransferTypeID       = calcSighash("global:swapout_transfer")        // 0x9152207ca5bb83bc
	SwapoutNativeTypeID         = calcSighash("global:swapout_native")          // 0x3b8e03e8d609f08f
	SkimLamportsTypeID          = calcSighash("global:skim_lamports")           // 0xff2ebac3ceab6f31
	ApplyMpcTypeID              = calcSighash("global:apply_mpc")
	EnableSwapTradeTypeID       = calcSighash("global:enable_swap_trade")
)

// SigHash the first 8 bytes to identify tx instruction
type SigHash [8]byte

// Uint64 to uint64
func (s *SigHash) Uint64() uint64 {
	return new(big.Int).SetBytes(s[:]).Uint64()
}

// SetUint64 set uint64
func (s *SigHash) SetUint64(i uint64) *SigHash {
	copy((*s)[:], new(big.Int).SetUint64(i).Bytes()[:])
	return s
}

func calcSighash(message string) (h SigHash) {
	hash := sha256.Sum256([]byte(message))
	copy(h[:], hash[:8])
	return h
}

// InitRouterProgram init router programID
func InitRouterProgram(programID types.PublicKey) {
	types.RegisterInstructionDecoder(programID, registryDecodeInstruction)
}

func registryDecodeInstruction(accounts []*types.AccountMeta, data []byte) (interface{}, error) {
	inst, err := DecodeInstruction(accounts, data)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

// DecodeInstruction decode instruction
func DecodeInstruction(accounts []*types.AccountMeta, data []byte) (*Instruction, error) {
	var inst Instruction
	if err := bin.NewDecoder(data).Decode(&inst); err != nil {
		return nil, fmt.Errorf("unable to decode instruction: %w", err)
	}

	if v, ok := inst.Impl.(types.AccountSettable); ok {
		err := v.SetAccounts(accounts)
		if err != nil {
			return nil, fmt.Errorf("unable to set accounts for instruction: %w", err)
		}
	}

	return &inst, nil
}

// InstructionDefVariant default variant
var InstructionDefVariant = NewVariantDefinition([]VariantType{
	{ID: SwapinMintTypeID, Name: "SwapinMint", Type: (*SwapinMint)(nil)},
	{ID: SwapinTransferTypeID, Name: "SwapinTransfer", Type: (*SwapinTransfer)(nil)},
	{ID: SwapinNativeTypeID, Name: "SwapinNative", Type: (*SwapinNative)(nil)},
})

// Instruction type
type Instruction struct {
	RouterProgramID types.PublicKey
	BaseVariant
}

// Accounts get accounts
func (i *Instruction) Accounts() (out []*types.AccountMeta) {
	switch i.TypeID {
	case SwapinMintTypeID:
		accounts := i.Impl.(*SwapinMint).Accounts
		out = []*types.AccountMeta{
			accounts.MPC,
			accounts.RouterAccount,
			accounts.To,
			accounts.TokenMint,
			accounts.TokenProgram,
		}
	case SwapinTransferTypeID:
		accounts := i.Impl.(*SwapinTransfer).Accounts
		out = []*types.AccountMeta{
			accounts.MPC,
			accounts.RouterAccount,
			accounts.From,
			accounts.To,
			accounts.TokenMint,
			accounts.TokenProgram,
		}
	case SwapinNativeTypeID:
		accounts := i.Impl.(*SwapinNative).Accounts
		out = []*types.AccountMeta{
			accounts.MPC,
			accounts.RouterAccount,
			accounts.To,
			accounts.SystemProgram,
		}
	case ChangeMpcTypeID:
		accounts := i.Impl.(*ChangeMpc).Accounts
		out = []*types.AccountMeta{
			accounts.MPC,
			accounts.RouterAccount,
			accounts.NewMPC,
		}
	case ApplyMpcTypeID:
		accounts := i.Impl.(*ChangeMpc).Accounts
		out = []*types.AccountMeta{
			accounts.MPC,
			accounts.RouterAccount,
			accounts.NewMPC,
		}
	case CreateAssociatedTokenTypeID:
		accounts := i.Impl.(*CreateAT).Accounts
		out = []*types.AccountMeta{
			accounts.Payer,
			accounts.Authority,
			accounts.Mint,
			accounts.AssociatedToken,
			accounts.Rent,
			accounts.SystemProgram,
			accounts.TokenProgram,
			accounts.AssociatedTokenProgram,
		}
	case EnableSwapTradeTypeID:
		accounts := i.Impl.(*EnableSwapTrade).Accounts
		out = []*types.AccountMeta{
			accounts.MPC,
			accounts.RouterAccount,
		}
	}
	return
}

// ProgramID get proram ID
func (i *Instruction) ProgramID() types.PublicKey {
	if i.RouterProgramID.IsZero() {
		panic("RouterProgramID is zero, please init it")
	}
	return i.RouterProgramID
}

// Data get data
func (i *Instruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := bin.NewEncoder(buf).Encode(i); err != nil {
		return nil, fmt.Errorf("unable to encode instruction: %w", err)
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary unmarshal binary
func (i *Instruction) UnmarshalBinary(decoder *bin.Decoder) (err error) {
	return i.BaseVariant.UnmarshalBinaryVariant(decoder, InstructionDefVariant)
}

// MarshalBinary marshal binary
func (i *Instruction) MarshalBinary(encoder *bin.Encoder) error {
	err := encoder.WriteUint64(i.TypeID.Uint64(), bin.BE())
	if err != nil {
		return fmt.Errorf("unable to write variant type: %w", err)
	}
	return encoder.Encode(i.Impl)
}

type ISwapinParams interface {
	GetSwapinParams() SwapinParams
}

// SwapinMint type
type SwapinMint struct {
	SwapinParams
	Accounts *SwapinMintAccounts `bin:"-"`
}

// SwapinMintAccounts type
type SwapinMintAccounts struct {
	MPC           *types.AccountMeta `text:"linear,notype"`
	RouterAccount *types.AccountMeta `text:"linear,notype"`
	To            *types.AccountMeta `text:"linear,notype"`
	TokenMint     *types.AccountMeta `text:"linear,notype"`
	TokenProgram  *types.AccountMeta `text:"linear,notype"`
}

func (i *SwapinMint) GetSwapinParams() SwapinParams {
	return i.SwapinParams
}

// NewSwapinMintInstruction new SwapinMint instruction
func NewSwapinMintInstruction(
	tx string, amount, fromChainID uint64,
	mpc, routerAccount, to, tokenMint, tokenProgram types.PublicKey,
) *Instruction {
	impl := &SwapinMint{
		SwapinParams: SwapinParams{
			Tx:          types.ToBorshString(tx),
			Amount:      amount,
			FromChainID: fromChainID,
		},
		Accounts: &SwapinMintAccounts{
			MPC:           &types.AccountMeta{PublicKey: mpc, IsWritable: true, IsSigner: true},
			RouterAccount: &types.AccountMeta{PublicKey: routerAccount},
			To:            &types.AccountMeta{PublicKey: to, IsWritable: true},
			TokenMint:     &types.AccountMeta{PublicKey: tokenMint, IsWritable: true},
			TokenProgram:  &types.AccountMeta{PublicKey: tokenProgram},
		},
	}
	return &Instruction{
		BaseVariant: BaseVariant{
			TypeID: SwapinMintTypeID,
			Impl:   impl,
		},
	}
}

// SetAccounts set accounts
func (i *SwapinMint) SetAccounts(accounts []*types.AccountMeta) error {
	i.Accounts = &SwapinMintAccounts{
		MPC:           accounts[0],
		RouterAccount: accounts[1],
		To:            accounts[2],
		TokenMint:     accounts[3],
		TokenProgram:  accounts[4],
	}
	return nil
}

// SwapinTransfer type
type SwapinTransfer struct {
	SwapinParams
	Accounts *SwapinTransferAccounts `bin:"-"`
}

func (i *SwapinTransfer) GetSwapinParams() SwapinParams {
	return i.SwapinParams
}

// SwapinTransferAccounts type
type SwapinTransferAccounts struct {
	MPC           *types.AccountMeta `text:"linear,notype"`
	RouterAccount *types.AccountMeta `text:"linear,notype"`
	From          *types.AccountMeta `text:"linear,notype"`
	To            *types.AccountMeta `text:"linear,notype"`
	TokenMint     *types.AccountMeta `text:"linear,notype"`
	TokenProgram  *types.AccountMeta `text:"linear,notype"`
}

// NewSwapinTransferInstruction new SwapinTransfer instruction
func NewSwapinTransferInstruction(
	tx string, amount, fromChainID uint64,
	mpc, routerAccount, from, to, tokenMint, tokenProgram types.PublicKey,
) *Instruction {
	impl := &SwapinTransfer{
		SwapinParams: SwapinParams{
			Tx:          types.ToBorshString(tx),
			Amount:      amount,
			FromChainID: fromChainID,
		},
		Accounts: &SwapinTransferAccounts{
			MPC:           &types.AccountMeta{PublicKey: mpc, IsWritable: true, IsSigner: true},
			RouterAccount: &types.AccountMeta{PublicKey: routerAccount},
			From:          &types.AccountMeta{PublicKey: from, IsWritable: true},
			To:            &types.AccountMeta{PublicKey: to, IsWritable: true},
			TokenMint:     &types.AccountMeta{PublicKey: tokenMint, IsWritable: true},
			TokenProgram:  &types.AccountMeta{PublicKey: tokenProgram},
		},
	}
	return &Instruction{
		BaseVariant: BaseVariant{
			TypeID: SwapinTransferTypeID,
			Impl:   impl,
		},
	}
}

// SetAccounts set accounts
func (i *SwapinTransfer) SetAccounts(accounts []*types.AccountMeta) error {
	i.Accounts = &SwapinTransferAccounts{
		MPC:           accounts[0],
		RouterAccount: accounts[1],
		From:          accounts[2],
		To:            accounts[3],
		TokenMint:     accounts[4],
		TokenProgram:  accounts[5],
	}
	return nil
}

// SwapinNative type
type SwapinNative struct {
	SwapinParams
	Accounts *SwapinNativeAccounts `bin:"-"`
}

func (i *SwapinNative) GetSwapinParams() SwapinParams {
	return i.SwapinParams
}

// SwapinNativeAccounts type
type SwapinNativeAccounts struct {
	MPC           *types.AccountMeta `text:"linear,notype"`
	RouterAccount *types.AccountMeta `text:"linear,notype"`
	To            *types.AccountMeta `text:"linear,notype"`
	SystemProgram *types.AccountMeta `text:"linear,notype"`
}

// NewSwapinNativeInstruction new SwapinNative instruction
func NewSwapinNativeInstruction(
	tx string, amount, fromChainID uint64,
	mpc, routerAccount, to, systemProgram types.PublicKey,
) *Instruction {
	impl := &SwapinNative{
		SwapinParams: SwapinParams{
			Tx:          types.ToBorshString(tx),
			Amount:      amount,
			FromChainID: fromChainID,
		},
		Accounts: &SwapinNativeAccounts{
			MPC:           &types.AccountMeta{PublicKey: mpc, IsWritable: true, IsSigner: true},
			RouterAccount: &types.AccountMeta{PublicKey: routerAccount, IsWritable: true},
			To:            &types.AccountMeta{PublicKey: to, IsWritable: true},
			SystemProgram: &types.AccountMeta{PublicKey: systemProgram},
		},
	}
	return &Instruction{
		BaseVariant: BaseVariant{
			TypeID: SwapinNativeTypeID,
			Impl:   impl,
		},
	}
}

// SetAccounts set accounts
func (i *SwapinNative) SetAccounts(accounts []*types.AccountMeta) error {
	i.Accounts = &SwapinNativeAccounts{
		MPC:           accounts[0],
		RouterAccount: accounts[1],
		To:            accounts[2],
		SystemProgram: accounts[3],
	}
	return nil
}

// SwapinParams swapin params
type SwapinParams struct {
	Tx          types.BorshString
	Amount      uint64
	FromChainID uint64
}

func (p *SwapinParams) String() string {
	return fmt.Sprintf("tx:%v amount:%v fromChainID:%v", p.Tx.String(), p.Amount, p.FromChainID)
}

// ChangeMpcParams type
type ChangeMpcParams struct {
	New *types.PublicKey `text:"linear,notype"`
}

// ChangeMpc type
type ChangeMpcAccounts struct {
	MPC           *types.AccountMeta `text:"linear,notype"`
	RouterAccount *types.AccountMeta `text:"linear,notype"`
	NewMPC        *types.AccountMeta `text:"linear,notype"`
}

// ChangeMpc type
type ChangeMpc struct {
	ChangeMpcParams
	Accounts *ChangeMpcAccounts `bin:"-"`
}

// SetAccounts set accounts
func (i *ChangeMpc) SetAccounts(accounts []*types.AccountMeta) error {
	i.Accounts = &ChangeMpcAccounts{
		MPC:           accounts[0],
		RouterAccount: accounts[1],
		NewMPC:        accounts[2],
	}
	return nil
}

func NewChangeMPCInstruction(
	mpc, routerAccount, newMpc types.PublicKey,
) *Instruction {
	impl := &ChangeMpc{
		ChangeMpcParams: ChangeMpcParams{
			New: &newMpc,
		},
		Accounts: &ChangeMpcAccounts{
			MPC:           &types.AccountMeta{PublicKey: mpc, IsWritable: true, IsSigner: true},
			RouterAccount: &types.AccountMeta{PublicKey: routerAccount, IsWritable: true},
			NewMPC:        &types.AccountMeta{PublicKey: newMpc},
		},
	}
	return &Instruction{
		BaseVariant: BaseVariant{
			TypeID: ChangeMpcTypeID,
			Impl:   impl,
		},
	}
}

func NewApplyMPCInstruction(
	mpc, routerAccount, newMpc types.PublicKey,
) *Instruction {
	impl := &ChangeMpc{
		ChangeMpcParams: ChangeMpcParams{
			New: &newMpc,
		},
		Accounts: &ChangeMpcAccounts{
			MPC:           &types.AccountMeta{PublicKey: mpc, IsWritable: true, IsSigner: true},
			RouterAccount: &types.AccountMeta{PublicKey: routerAccount, IsWritable: true},
			NewMPC:        &types.AccountMeta{PublicKey: newMpc},
		},
	}
	return &Instruction{
		BaseVariant: BaseVariant{
			TypeID: ApplyMpcTypeID,
			Impl:   impl,
		},
	}
}

// ChangeMpc type
type CreateATAccounts struct {
	Payer                  *types.AccountMeta `text:"linear,notype"`
	Authority              *types.AccountMeta `text:"linear,notype"`
	Mint                   *types.AccountMeta `text:"linear,notype"`
	AssociatedToken        *types.AccountMeta `text:"linear,notype"`
	Rent                   *types.AccountMeta `text:"linear,notype"`
	SystemProgram          *types.AccountMeta `text:"linear,notype"`
	TokenProgram           *types.AccountMeta `text:"linear,notype"`
	AssociatedTokenProgram *types.AccountMeta `text:"linear,notype"`
}

// ChangeMpc type
type CreateAT struct {
	Accounts *CreateATAccounts `bin:"-"`
}

// SetAccounts set accounts
func (i *CreateAT) SetAccounts(accounts []*types.AccountMeta) error {
	i.Accounts = &CreateATAccounts{
		Payer:                  accounts[0],
		Authority:              accounts[1],
		Mint:                   accounts[2],
		AssociatedToken:        accounts[3],
		Rent:                   accounts[4],
		SystemProgram:          accounts[5],
		TokenProgram:           accounts[6],
		AssociatedTokenProgram: accounts[7],
	}
	return nil
}

func NewCreateATAInstruction(
	payer, owner, tokenProgramID, ownertokenATA types.PublicKey,
) *Instruction {
	impl := &CreateAT{
		Accounts: &CreateATAccounts{
			Payer:                  &types.AccountMeta{PublicKey: payer, IsWritable: true, IsSigner: true},
			Authority:              &types.AccountMeta{PublicKey: owner},
			Mint:                   &types.AccountMeta{PublicKey: tokenProgramID},
			AssociatedToken:        &types.AccountMeta{PublicKey: ownertokenATA, IsWritable: true},
			Rent:                   &types.AccountMeta{PublicKey: system.SysvarRentProgramID},
			SystemProgram:          &types.AccountMeta{PublicKey: system.SystemProgramID},
			TokenProgram:           &types.AccountMeta{PublicKey: types.TokenProgramID},
			AssociatedTokenProgram: &types.AccountMeta{PublicKey: types.ATAProgramID},
		},
	}
	return &Instruction{
		BaseVariant: BaseVariant{
			TypeID: CreateAssociatedTokenTypeID,
			Impl:   impl,
		},
	}
}

type EnableSwapTradeParams struct {
	Enable bool `text:"linear,notype"`
}

type EnableSwapTradeAccounts struct {
	MPC           *types.AccountMeta `text:"linear,notype"`
	RouterAccount *types.AccountMeta `text:"linear,notype"`
}

type EnableSwapTrade struct {
	EnableSwapTradeParams
	Accounts *EnableSwapTradeAccounts `bin:"-"`
}

func (i *EnableSwapTrade) SetAccounts(accounts []*types.AccountMeta) error {
	i.Accounts = &EnableSwapTradeAccounts{
		MPC:           accounts[0],
		RouterAccount: accounts[1],
	}
	return nil
}

func NewEnableSwapTradeInstruction(
	enable bool, mpc, routerAccount types.PublicKey,
) *Instruction {
	impl := &EnableSwapTrade{
		EnableSwapTradeParams: EnableSwapTradeParams{
			Enable: enable,
		},
		Accounts: &EnableSwapTradeAccounts{
			MPC:           &types.AccountMeta{PublicKey: mpc, IsWritable: true, IsSigner: true},
			RouterAccount: &types.AccountMeta{PublicKey: routerAccount},
		},
	}
	return &Instruction{
		BaseVariant: BaseVariant{
			TypeID: EnableSwapTradeTypeID,
			Impl:   impl,
		},
	}
}
