package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/Dharitri-org/me-vm-common"
	"github.com/Dharitri-org/me-vm-common/check"
	"github.com/Dharitri-org/me-vm-common/data/dct"
	"github.com/Dharitri-org/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createDCTNFTMultiTransferWithStubArguments() *dctNFTMultiTransfer {
	multiTransfer, _ := NewDCTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
	)

	return multiTransfer
}

func createDCTNFTMultiTransferWithMockArguments(selfShard uint32, numShards uint32, pauseHandler vmcommon.DCTPauseHandler) *dctNFTMultiTransfer {
	marshalizer := &mock.MarshalizerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(numShards)
	shardCoordinator.CurrentShard = selfShard
	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		lastByte := uint32(address[len(address)-1])
		return lastByte
	}
	mapAccounts := make(map[string]vmcommon.UserAccountHandler)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
		GetExistingAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
	}

	multiTransfer, _ := NewDCTNFTMultiTransferFunc(
		1,
		marshalizer,
		pauseHandler,
		accounts,
		shardCoordinator,
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
	)

	return multiTransfer
}

func TestNewDCTNFTMultiTransferFunc_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	multiTransfer, err := NewDCTNFTMultiTransferFunc(
		0,
		nil,
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilMarshalizer, err)

	multiTransfer, err = NewDCTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		nil,
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilPauseHandler, err)

	multiTransfer, err = NewDCTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		nil,
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilAccountsAdapter, err)

	multiTransfer, err = NewDCTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		nil,
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilShardCoordinator, err)

	multiTransfer, err = NewDCTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		nil,
	)
	assert.True(t, check.IfNil(multiTransfer))
	assert.Equal(t, ErrNilEpochHandler, err)
}

func TestNewDCTNFTMultiTransferFunc(t *testing.T) {
	t.Parallel()

	multiTransfer, err := NewDCTNFTMultiTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		0,
		&mock.EpochNotifierStub{},
	)
	assert.False(t, check.IfNil(multiTransfer))
	assert.Nil(t, err)
}

func TestDCTNFTMultiTransfer_SetPayable(t *testing.T) {
	t.Parallel()

	multiTransfer := createDCTNFTMultiTransferWithStubArguments()
	err := multiTransfer.SetPayableHandler(nil)
	assert.Equal(t, ErrNilPayableHandler, err)

	handler := &mock.PayableHandlerStub{}
	err = multiTransfer.SetPayableHandler(handler)
	assert.Nil(t, err)
	assert.True(t, handler == multiTransfer.payableHandler) //pointer testing
}

func TestDCTNFTMultiTransfer_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	multiTransfer := createDCTNFTMultiTransferWithStubArguments()
	multiTransfer.SetNewGasConfig(nil)
	assert.Equal(t, uint64(0), multiTransfer.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, multiTransfer.gasConfig)

	gasCost := createMockGasCost()
	multiTransfer.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.DCTNFTMultiTransfer, multiTransfer.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, multiTransfer.gasConfig)
}

func TestDCTNFTMultiTransfer_ProcessBuiltinFunctionInvalidArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	multiTransfer := createDCTNFTMultiTransferWithStubArguments()
	vmOutput, err := multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	}
	vmOutput, err = multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidArguments, err)

	multiTransfer.shardCoordinator = &mock.ShardCoordinatorStub{ComputeIdCalled: func(address []byte) uint32 {
		return vmcommon.MetachainShardId
	}}

	token1 := []byte("token")
	senderAddress := bytes.Repeat([]byte{2}, 32)
	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{vmcommon.DCTSCAddress, big.NewInt(1).Bytes(), token1, big.NewInt(1).Bytes(), big.NewInt(1).Bytes()},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmOutput, err = multiTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)
}

func TestDCTNFTMultiTransfer_ProcessBuiltinFunctionOnSameShardWithScCall(t *testing.T) {
	t.Parallel()

	multiTransfer := createDCTNFTMultiTransferWithMockArguments(0, 1, &mock.PauseHandlerStub{})
	_ = multiTransfer.SetPayableHandler(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		})
	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token1")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCTNFTToken(token1, vmcommon.NonFungible, tokenNonce, initialTokens, multiTransfer.marshalizer, sender.(vmcommon.UserAccountHandler))
	createDCTNFTToken(token2, vmcommon.Fungible, 0, initialTokens, multiTransfer.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = multiTransfer.accounts.SaveAccount(sender)
	_ = multiTransfer.accounts.SaveAccount(destination)
	_, _ = multiTransfer.accounts.Commit()

	//reload accounts
	sender, err = multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := multiTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransfer.accounts.SaveAccount(sender)
	_, _ = multiTransfer.accounts.Commit()

	//reload accounts
	sender, err = multiTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = multiTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransfer.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) //3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransfer.marshalizer, sender, token2, 0, big.NewInt(2))
	testNFTTokenShouldExist(t, multiTransfer.marshalizer, destination, token1, tokenNonce, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransfer.marshalizer, destination, token2, 0, big.NewInt(1))
	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestDCTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationDoesNotHoldingNFTWithSCCall(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createDCTNFTMultiTransferWithMockArguments(1, 2, &mock.PauseHandlerStub{})
	_ = multiTransferSenderShard.SetPayableHandler(payableHandler)

	multiTransferDestinationShard := createDCTNFTMultiTransferWithMockArguments(0, 2, &mock.PauseHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token1")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCTNFTToken(token1, vmcommon.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	createDCTNFTToken(token2, vmcommon.Fungible, 0, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	//reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1000000,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	//reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) //3 initial - 1 transferred

	funcName, args := extractScResultsFromVmOutput(t, vmOutput)

	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token1, tokenNonce, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token2, 0, big.NewInt(1))
	funcName, args = extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestDCTNFTMultiTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationHoldsNFT(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	multiTransferSenderShard := createDCTNFTMultiTransferWithMockArguments(0, 2, &mock.PauseHandlerStub{})
	_ = multiTransferSenderShard.SetPayableHandler(payableHandler)

	multiTransferDestinationShard := createDCTNFTMultiTransferWithMockArguments(1, 2, &mock.PauseHandlerStub{})
	_ = multiTransferDestinationShard.SetPayableHandler(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token1")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCTNFTToken(token1, vmcommon.NonFungible, tokenNonce, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	createDCTNFTToken(token2, vmcommon.Fungible, 0, initialTokens, multiTransferSenderShard.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	//reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := multiTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = multiTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = multiTransferSenderShard.accounts.Commit()

	//reload sender account
	sender, err = multiTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token1, tokenNonce, big.NewInt(2)) //3 initial - 1 transferred
	testNFTTokenShouldExist(t, multiTransferSenderShard.marshalizer, sender, token2, 0, big.NewInt(2))
	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destinationNumTokens := big.NewInt(1000)
	destination, err := multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createDCTNFTToken(token1, vmcommon.NonFungible, tokenNonce, destinationNumTokens, multiTransferDestinationShard.marshalizer, destination.(vmcommon.UserAccountHandler))
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = multiTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = multiTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = multiTransferDestinationShard.accounts.Commit()

	destination, err = multiTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	expected := big.NewInt(0).Add(destinationNumTokens, big.NewInt(1))
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token1, tokenNonce, expected)
	testNFTTokenShouldExist(t, multiTransferDestinationShard.marshalizer, destination, token2, 0, big.NewInt(1))
}

func TestDCTNFTMultiTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	transferFunc := createDCTNFTMultiTransferWithMockArguments(0, 1, &mock.PauseHandlerStub{})
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress[31] = 0
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token1")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCTNFTToken(token1, vmcommon.NonFungible, tokenNonce, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	createDCTNFTToken(token2, vmcommon.Fungible, 0, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	dctFrozen := DCTUserMetadata{Frozen: true}

	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	//reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 100000,
		},
		RecipientAddr: senderAddress,
	}

	destination, _ := transferFunc.accounts.LoadAccount(destinationAddress)
	tokenId := append(keyPrefix, token1...)
	dctKey := computeDCTNFTTokenKey(tokenId, tokenNonce)
	dctToken := &dct.DCToken{Value: big.NewInt(0), Properties: dctFrozen.ToBytes()}
	marshaledData, _ := transferFunc.marshalizer.Marshal(dctToken)
	_ = destination.(vmcommon.UserAccountHandler).AccountDataHandler().SaveKeyValue(dctKey, marshaledData)
	_ = transferFunc.accounts.SaveAccount(destination)
	_, _ = transferFunc.accounts.Commit()

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, ErrDCTIsFrozenForAccount, err)

	vmInput.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Nil(t, err)
}

func TestDCTNFTMultiTransfer_NotEnoughGas(t *testing.T) {
	t.Parallel()

	transferFunc := createDCTNFTMultiTransferWithMockArguments(0, 1, &mock.PauseHandlerStub{})
	_ = transferFunc.SetPayableHandler(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	token1 := []byte("token1")
	token2 := []byte("token1")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCTNFTToken(token1, vmcommon.NonFungible, tokenNonce, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	createDCTNFTToken(token2, vmcommon.Fungible, 0, initialTokens, transferFunc.marshalizer, sender.(vmcommon.UserAccountHandler))
	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	//reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{destinationAddress, big.NewInt(2).Bytes(), token1, nonceBytes, quantityBytes, token2, big.NewInt(0).Bytes(), quantityBytes},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), sender.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, err, ErrNotEnoughGas)
}
