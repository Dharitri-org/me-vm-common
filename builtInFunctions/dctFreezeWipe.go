package builtInFunctions

import (
	"bytes"
	"math/big"

	vmcommon "github.com/Dharitri-org/me-vm-common"
	"github.com/Dharitri-org/me-vm-common/check"
)

type dctFreezeWipe struct {
	baseAlwaysActive
	marshalizer vmcommon.Marshalizer
	keyPrefix   []byte
	wipe        bool
	freeze      bool
}

// NewDCTFreezeWipeFunc returns the dct freeze/un-freeze/wipe built-in function component
func NewDCTFreezeWipeFunc(
	marshalizer vmcommon.Marshalizer,
	freeze bool,
	wipe bool,
) (*dctFreezeWipe, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}

	e := &dctFreezeWipe{
		marshalizer: marshalizer,
		keyPrefix:   []byte(vmcommon.DharitriProtectedKeyPrefix + vmcommon.DCTKeyIdentifier),
		freeze:      freeze,
		wipe:        wipe,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctFreezeWipe) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCT transfer function call
func (e *dctFreezeWipe) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, vmcommon.DCTSCAddress) {
		return nil, ErrAddressIsNotDCTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	dctTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	if e.wipe {
		err := e.wipeIfApplicable(acntDst, dctTokenKey)
		if err != nil {
			return nil, err
		}
	} else {
		err := e.toggleFreeze(acntDst, dctTokenKey)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	if e.wipe {
		addDCTEntryInVMOutput(vmOutput, []byte(vmcommon.BuiltInFunctionDCTWipe), vmInput.Arguments[0], big.NewInt(0), vmInput.CallerAddr, acntDst.AddressBytes())
	}

	return vmOutput, nil
}

func (e *dctFreezeWipe) wipeIfApplicable(acntDst vmcommon.UserAccountHandler, tokenKey []byte) error {
	tokenData, err := getDCTDataFromKey(acntDst, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	dctUserMetadata := DCTUserMetadataFromBytes(tokenData.Properties)
	if !dctUserMetadata.Frozen {
		return ErrCannotWipeAccountNotFrozen
	}

	return acntDst.AccountDataHandler().SaveKeyValue(tokenKey, nil)
}

func (e *dctFreezeWipe) toggleFreeze(acntDst vmcommon.UserAccountHandler, tokenKey []byte) error {
	tokenData, err := getDCTDataFromKey(acntDst, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	dctUserMetadata := DCTUserMetadataFromBytes(tokenData.Properties)
	dctUserMetadata.Frozen = e.freeze
	tokenData.Properties = dctUserMetadata.ToBytes()

	err = saveDCTData(acntDst, tokenData, tokenKey, e.marshalizer)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctFreezeWipe) IsInterfaceNil() bool {
	return e == nil
}
