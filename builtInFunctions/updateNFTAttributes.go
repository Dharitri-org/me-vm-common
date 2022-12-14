package builtInFunctions

import (
	"math/big"
	"sync"

	vmcommon "github.com/Dharitri-org/me-vm-common"
	"github.com/Dharitri-org/me-vm-common/atomic"
	"github.com/Dharitri-org/me-vm-common/check"
)

type dctNFTupdate struct {
	*baseEnabled
	keyPrefix    []byte
	marshalizer  vmcommon.Marshalizer
	pauseHandler vmcommon.DCTPauseHandler
	rolesHandler vmcommon.DCTRoleHandler
	gasConfig    vmcommon.BaseOperationCost
	funcGasCost  uint64
	mutExecution sync.RWMutex
}

// NewDCTNFTAddUriFunc returns the dct NFT update attribute built-in function component
func NewDCTNFTUpdateAttributesFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	marshalizer vmcommon.Marshalizer,
	pauseHandler vmcommon.DCTPauseHandler,
	rolesHandler vmcommon.DCTRoleHandler,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*dctNFTupdate, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(pauseHandler) {
		return nil, ErrNilPauseHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &dctNFTupdate{
		keyPrefix:    []byte(vmcommon.DharitriProtectedKeyPrefix + vmcommon.DCTKeyIdentifier),
		marshalizer:  marshalizer,
		funcGasCost:  funcGasCost,
		mutExecution: sync.RWMutex{},
		pauseHandler: pauseHandler,
		gasConfig:    gasConfig,
		rolesHandler: rolesHandler,
	}

	e.baseEnabled = &baseEnabled{
		function:        vmcommon.BuiltInFunctionDCTNFTUpdateAttributes,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctNFTupdate) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCTNFTUpdateAttributes
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCT NFT add quantity function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - new attributes
func (e *dctNFTupdate) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkDCTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) != 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(vmcommon.DCTRoleNFTUpdateAttributes))
	if err != nil {
		return nil, err
	}

	gasCostForStore := uint64(len(vmInput.Arguments[2])) * e.gasConfig.StorePerByte
	if vmInput.GasProvided < e.funcGasCost+gasCostForStore {
		return nil, ErrNotEnoughGas
	}

	dctTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}
	dctData, err := getDCTNFTTokenOnSender(acntSnd, dctTokenKey, nonce, e.marshalizer)
	if err != nil {
		return nil, err
	}

	dctData.TokenMetaData.Attributes = vmInput.Arguments[2]

	_, err = saveDCTNFTToken(acntSnd, dctTokenKey, dctData, e.marshalizer, e.pauseHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	logEntry := newEntryForNFT(vmcommon.BuiltInFunctionDCTNFTUpdateAttributes, vmInput.CallerAddr, vmInput.Arguments[0], nonce)
	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost - gasCostForStore,
		Logs:         []*vmcommon.LogEntry{logEntry},
	}
	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctNFTupdate) IsInterfaceNil() bool {
	return e == nil
}
