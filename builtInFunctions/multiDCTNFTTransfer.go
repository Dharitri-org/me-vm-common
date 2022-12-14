package builtInFunctions

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sync"

	vmcommon "github.com/Dharitri-org/me-vm-common"
	"github.com/Dharitri-org/me-vm-common/atomic"
	"github.com/Dharitri-org/me-vm-common/check"
	"github.com/Dharitri-org/me-vm-common/data/dct"
)

type dctNFTMultiTransfer struct {
	*baseEnabled
	keyPrefix        []byte
	marshalizer      vmcommon.Marshalizer
	pauseHandler     vmcommon.DCTPauseHandler
	payableHandler   vmcommon.PayableHandler
	funcGasCost      uint64
	accounts         vmcommon.AccountsAdapter
	shardCoordinator vmcommon.Coordinator
	gasConfig        vmcommon.BaseOperationCost
	mutExecution     sync.RWMutex
}

const argumentsPerTransfer = uint64(3)

// NewDCTNFTMultiTransferFunc returns the dct NFT multi transfer built-in function component
func NewDCTNFTMultiTransferFunc(
	funcGasCost uint64,
	marshalizer vmcommon.Marshalizer,
	pauseHandler vmcommon.DCTPauseHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
	gasConfig vmcommon.BaseOperationCost,
	activationEpoch uint32,
	epochNotifier vmcommon.EpochNotifier,
) (*dctNFTMultiTransfer, error) {
	if check.IfNil(marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(pauseHandler) {
		return nil, ErrNilPauseHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(epochNotifier) {
		return nil, ErrNilEpochHandler
	}

	e := &dctNFTMultiTransfer{
		keyPrefix:        []byte(vmcommon.DharitriProtectedKeyPrefix + vmcommon.DCTKeyIdentifier),
		marshalizer:      marshalizer,
		pauseHandler:     pauseHandler,
		funcGasCost:      funcGasCost,
		accounts:         accounts,
		shardCoordinator: shardCoordinator,
		gasConfig:        gasConfig,
		mutExecution:     sync.RWMutex{},
		payableHandler:   &disabledPayableHandler{},
	}

	e.baseEnabled = &baseEnabled{
		function:        vmcommon.BuiltInFunctionMultiDCTNFTTransfer,
		activationEpoch: activationEpoch,
		flagActivated:   atomic.Flag{},
	}

	epochNotifier.RegisterNotifyHandler(e)

	return e, nil
}

// SetPayableHandler will set the payable handler to the function
func (e *dctNFTMultiTransfer) SetPayableHandler(payableHandler vmcommon.PayableHandler) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctNFTMultiTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCTNFTMultiTransfer
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCT NFT transfer roles function call
// Requires the following arguments:
// arg0 - destination address
// arg1 - number of tokens to transfer
// list of (tokenID - nonce - quantity) - in case of DCT nonce == 0
// function and list of arguments for SC Call
// if cross-shard, the rest of arguments will be filled inside the SCR
// arg0 - number of tokens to transfer
// list of (tokenID - nonce - quantity/DCT NFT data)
// function and list of arguments for SC Call
func (e *dctNFTMultiTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicDCTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 5 {
		return nil, ErrInvalidArguments
	}

	if bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return e.processDCTNFTMultiTransferOnSenderShard(acntSnd, vmInput)
	}

	// in cross shard NFT transfer the sender account must be nil
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidRcvAddr
	}
	if check.IfNil(acntDst) {
		return nil, ErrInvalidRcvAddr
	}

	numOfTransfers := big.NewInt(0).SetBytes(vmInput.Arguments[0]).Uint64()
	if numOfTransfers == 0 {
		return nil, fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 1
	if uint64(len(vmInput.Arguments)) < minNumOfArguments {
		return nil, fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	vmOutput := &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided}
	vmOutput.Logs = make([]*vmcommon.LogEntry, numOfTransfers)
	startIndex := uint64(1)
	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		tokenID := vmInput.Arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+1]).Uint64()

		dctTokenKey := append(e.keyPrefix, tokenID...)

		if nonce > 0 {
			marshaledNFTTransfer := vmInput.Arguments[tokenStartIndex+2]
			dctTransferData := &dct.DCToken{}
			err = e.marshalizer.Unmarshal(dctTransferData, marshaledNFTTransfer)
			if err != nil {
				return nil, err
			}

			err = e.addNFTToDestination(
				vmInput.RecipientAddr,
				acntDst,
				dctTransferData,
				dctTokenKey,
				mustVerifyPayable(vmInput, int(minNumOfArguments)),
				vmInput.ReturnCallAfterError)
			if err != nil {
				return nil, err
			}
		} else {
			err = addToDCTBalance(acntDst, dctTokenKey, big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2]), e.marshalizer, e.pauseHandler, vmInput.ReturnCallAfterError)
			if err != nil {
				return nil, err
			}
		}

		logEntry := newEntryForNFT(vmcommon.BuiltInFunctionMultiDCTNFTTransfer, vmInput.CallerAddr, tokenID, nonce)
		logEntry.Topics = append(logEntry.Topics, acntDst.AddressBytes())
		vmOutput.Logs[i] = logEntry
	}

	// no need to consume gas on destination - sender already paid for it
	if len(vmInput.Arguments) > int(minNumOfArguments) && vmcommon.IsSmartContractAddress(vmInput.RecipientAddr) {
		var callArgs [][]byte
		if len(vmInput.Arguments) > int(minNumOfArguments)+1 {
			callArgs = vmInput.Arguments[minNumOfArguments+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[minNumOfArguments]),
			callArgs,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return vmOutput, nil
}

func (e *dctNFTMultiTransfer) processDCTNFTMultiTransferOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	dstAddress := vmInput.Arguments[0]
	if len(dstAddress) != len(vmInput.CallerAddr) {
		return nil, fmt.Errorf("%w, not a valid destination address", ErrInvalidArguments)
	}
	if bytes.Equal(dstAddress, vmInput.CallerAddr) {
		return nil, fmt.Errorf("%w, can not transfer to self", ErrInvalidArguments)
	}
	if e.shardCoordinator.ComputeId(dstAddress) == vmcommon.MetachainShardId {
		return nil, ErrInvalidRcvAddr
	}
	numOfTransfers := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if numOfTransfers == 0 {
		return nil, fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 2
	if uint64(len(vmInput.Arguments)) < minNumOfArguments {
		return nil, fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	multiTransferCost := numOfTransfers * e.funcGasCost
	if vmInput.GasProvided < multiTransferCost {
		return nil, ErrNotEnoughGas
	}

	verifyPayable := mustVerifyPayable(vmInput, int(minNumOfArguments))
	acntDst, err := e.loadAccountIfInShard(dstAddress)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - multiTransferCost,
		Logs:         make([]*vmcommon.LogEntry, numOfTransfers),
	}

	startIndex := uint64(2)
	listDctData := make([]*dct.DCToken, numOfTransfers)
	listTokenID := make([][]byte, numOfTransfers)
	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		listTokenID[i] = vmInput.Arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+1]).Uint64()
		quantityToTransfer := big.NewInt(0).SetBytes(vmInput.Arguments[tokenStartIndex+2])
		listDctData[i], err = e.transferOneTokenOnSenderShard(
			acntSnd,
			acntDst,
			dstAddress,
			listTokenID[i],
			nonce,
			quantityToTransfer,
			verifyPayable,
			vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}

		logEntry := newEntryForNFT(vmcommon.BuiltInFunctionMultiDCTNFTTransfer, vmInput.CallerAddr, listTokenID[i], nonce)
		logEntry.Topics = append(logEntry.Topics, dstAddress)
		vmOutput.Logs[i] = logEntry
	}

	if !check.IfNil(acntDst) {
		err = e.accounts.SaveAccount(acntDst)
		if err != nil {
			return nil, err
		}
	}

	err = e.createDCTNFTOutputTransfers(vmInput, vmOutput, listDctData, listTokenID, dstAddress)
	if err != nil {
		return nil, err
	}

	return vmOutput, nil
}

func (e *dctNFTMultiTransfer) transferOneTokenOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	acntDst vmcommon.UserAccountHandler,
	dstAddress []byte,
	tokenID []byte,
	nonce uint64,
	quantityToTransfer *big.Int,
	verifyPayable bool,
	isReturnCallWithError bool,
) (*dct.DCToken, error) {
	if quantityToTransfer.Cmp(zero) <= 0 {
		return nil, ErrInvalidNFTQuantity
	}

	dctTokenKey := append(e.keyPrefix, tokenID...)
	dctData, err := getDCTNFTTokenOnSender(acntSnd, dctTokenKey, nonce, e.marshalizer)
	if err != nil {
		return nil, err
	}

	if dctData.Value.Cmp(quantityToTransfer) < 0 {
		return nil, ErrInvalidNFTQuantity
	}
	dctData.Value.Sub(dctData.Value, quantityToTransfer)

	_, err = saveDCTNFTToken(acntSnd, dctTokenKey, dctData, e.marshalizer, e.pauseHandler, isReturnCallWithError)
	if err != nil {
		return nil, err
	}

	dctData.Value.Set(quantityToTransfer)

	if !check.IfNil(acntDst) {
		err = e.addNFTToDestination(dstAddress, acntDst, dctData, dctTokenKey, verifyPayable, isReturnCallWithError)
		if err != nil {
			return nil, err
		}
	}

	return dctData, nil
}

func (e *dctNFTMultiTransfer) loadAccountIfInShard(dstAddress []byte) (vmcommon.UserAccountHandler, error) {
	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		return nil, nil
	}

	accountHandler, errLoad := e.accounts.LoadAccount(dstAddress)
	if errLoad != nil {
		return nil, errLoad
	}
	userAccount, ok := accountHandler.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAccount, nil
}

func (e *dctNFTMultiTransfer) createDCTNFTOutputTransfers(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	listDCTTransferData []*dct.DCToken,
	listTokenIDs [][]byte,
	dstAddress []byte,
) error {
	multiTransferCallArgs := make([][]byte, 0, argumentsPerTransfer*uint64(len(listDCTTransferData))+1)
	numTokenTransfer := big.NewInt(int64(len(listDCTTransferData))).Bytes()
	multiTransferCallArgs = append(multiTransferCallArgs, numTokenTransfer)

	for i, dctTransferData := range listDCTTransferData {
		multiTransferCallArgs = append(multiTransferCallArgs, listTokenIDs[i])
		if dctTransferData.TokenMetaData != nil {
			marshaledNFTTransfer, err := e.marshalizer.Marshal(dctTransferData)
			if err != nil {
				return err
			}

			gasForTransfer := uint64(len(marshaledNFTTransfer)) * e.gasConfig.DataCopyPerByte
			if gasForTransfer > vmOutput.GasRemaining {
				return ErrNotEnoughGas
			}
			vmOutput.GasRemaining -= gasForTransfer

			multiTransferCallArgs = append(multiTransferCallArgs, big.NewInt(0).SetUint64(dctTransferData.TokenMetaData.Nonce).Bytes())
			multiTransferCallArgs = append(multiTransferCallArgs, marshaledNFTTransfer)
		} else {
			multiTransferCallArgs = append(multiTransferCallArgs, []byte{0})
			multiTransferCallArgs = append(multiTransferCallArgs, dctTransferData.Value.Bytes())
		}
	}

	minNumOfArguments := uint64(len(listDCTTransferData))*argumentsPerTransfer + 2
	if uint64(len(vmInput.Arguments)) > minNumOfArguments {
		multiTransferCallArgs = append(multiTransferCallArgs, vmInput.Arguments[minNumOfArguments:]...)
	}

	isSCCallAfter := uint64(len(vmInput.Arguments)) > minNumOfArguments && vmcommon.IsSmartContractAddress(dstAddress)

	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		gasToTransfer := uint64(0)
		if isSCCallAfter {
			gasToTransfer = vmOutput.GasRemaining
			vmOutput.GasRemaining = 0
		}
		addNFTTransferToVMOutput(
			vmInput.CallerAddr,
			dstAddress,
			vmcommon.BuiltInFunctionMultiDCTNFTTransfer,
			multiTransferCallArgs,
			vmInput.GasLocked,
			gasToTransfer,
			vmInput.CallType,
			vmOutput,
		)

		return nil
	}

	if isSCCallAfter {
		var callArgs [][]byte
		if uint64(len(vmInput.Arguments)) > minNumOfArguments+1 {
			callArgs = vmInput.Arguments[minNumOfArguments+1:]
		}

		addOutputTransferToVMOutput(
			vmInput.CallerAddr,
			string(vmInput.Arguments[minNumOfArguments]),
			callArgs,
			dstAddress,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return nil
}

func (e *dctNFTMultiTransfer) addNFTToDestination(
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	dctDataToTransfer *dct.DCToken,
	dctTokenKey []byte,
	mustVerifyPayable bool,
	isReturnCallWithError bool,
) error {
	if mustVerifyPayable {
		isPayable, errIsPayable := e.payableHandler.IsPayable(dstAddress)
		if errIsPayable != nil {
			return errIsPayable
		}
		if !isPayable {
			return ErrAccountNotPayable
		}
	}

	nonce := uint64(0)
	if dctDataToTransfer.TokenMetaData != nil {
		nonce = dctDataToTransfer.TokenMetaData.Nonce
	}

	currentDCTData, _, err := getDCTNFTTokenOnDestination(userAccount, dctTokenKey, nonce, e.marshalizer)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, dctTokenKey, currentDCTData, e.pauseHandler, isReturnCallWithError)
	if err != nil {
		return err
	}

	if currentDCTData.TokenMetaData != nil {
		if !bytes.Equal(currentDCTData.TokenMetaData.Hash, dctDataToTransfer.TokenMetaData.Hash) {
			return ErrWrongNFTOnDestination
		}
		dctDataToTransfer.Value.Add(dctDataToTransfer.Value, currentDCTData.Value)
	}

	_, err = saveDCTNFTToken(userAccount, dctTokenKey, dctDataToTransfer, e.marshalizer, e.pauseHandler, isReturnCallWithError)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctNFTMultiTransfer) IsInterfaceNil() bool {
	return e == nil
}
