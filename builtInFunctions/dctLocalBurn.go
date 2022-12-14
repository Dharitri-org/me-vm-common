package builtInFunctions

import (
	"bytes"
	"math/big"
	"sync"

	vmcommon "github.com/Dharitri-org/me-vm-common"
	"github.com/Dharitri-org/me-vm-common/check"
)

type dctLocalBurn struct {
	baseAlwaysActive
	keyPrefix    []byte
	marshalizer  vmcommon.Marshalizer
	pauseHandler vmcommon.DCTPauseHandler
	rolesHandler vmcommon.DCTRoleHandler
	funcGasCost  uint64
	mutExecution sync.RWMutex
}

// NewDCTLocalBurnFunc returns the dct local burn built-in function component
func NewDCTLocalBurnFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	pauseHandler vmcommon.DCTPauseHandler,
	rolesHandler vmcommon.DCTRoleHandler,
) (*dctLocalBurn, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(pauseHandler) {
		return nil, ErrNilPauseHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}

	e := &dctLocalBurn{
		keyPrefix:    []byte(vmcommon.DharitriProtectedKeyPrefix + vmcommon.DCTKeyIdentifier),
		marshalizer:  marshalizer,
		pauseHandler: pauseHandler,
		rolesHandler: rolesHandler,
		funcGasCost:  funcGasCost,
		mutExecution: sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctLocalBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCTLocalBurn
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCT local burn function call
func (e *dctLocalBurn) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkInputArgumentsForLocalAction(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}

	tokenID := vmInput.Arguments[0]
	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(vmcommon.DCTRoleLocalBurn))
	if err != nil {
		return nil, err
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	dctTokenKey := append(e.keyPrefix, tokenID...)
	err = addToDCTBalance(acntSnd, dctTokenKey, big.NewInt(0).Neg(value), e.marshalizer, e.pauseHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - e.funcGasCost}

	addDCTEntryInVMOutput(vmOutput, []byte(vmcommon.BuiltInFunctionDCTLocalBurn), vmInput.Arguments[0], value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctLocalBurn) IsInterfaceNil() bool {
	return e == nil
}

func checkBasicDCTArguments(vmInput *vmcommon.ContractCallInput) error {
	if vmInput == nil {
		return ErrNilVmInput
	}
	if vmInput.CallValue == nil {
		return ErrNilValue
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) < vmcommon.MinLenArgumentsDCTTransfer {
		return ErrInvalidArguments
	}
	return nil
}

func checkInputArgumentsForLocalAction(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	funcGasCost uint64,
) error {
	err := checkBasicDCTArguments(vmInput)
	if err != nil {
		return err
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return ErrInvalidRcvAddr
	}
	if check.IfNil(acntSnd) {
		return ErrNilUserAccount
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return ErrNegativeValue
	}
	if vmInput.GasProvided < funcGasCost {
		return ErrNotEnoughGas
	}

	return nil
}
