package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/Dharitri-org/me-vm-common"
	"github.com/Dharitri-org/me-vm-common/check"
	"github.com/Dharitri-org/me-vm-common/data/dct"
	"github.com/Dharitri-org/me-vm-common/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createNftCreateWithStubArguments() *dctNFTCreate {
	nftCreate, _ := NewDCTNFTCreateFunc(
		1,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		&mock.DCTRoleHandlerStub{},
	)

	return nftCreate
}

func TestNewDCTNFTCreateFunc_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	nftCreate, err := NewDCTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		nil,
		&mock.PauseHandlerStub{},
		&mock.DCTRoleHandlerStub{},
	)
	assert.True(t, check.IfNil(nftCreate))
	assert.Equal(t, ErrNilMarshalizer, err)

	nftCreate, err = NewDCTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		nil,
		&mock.DCTRoleHandlerStub{},
	)
	assert.True(t, check.IfNil(nftCreate))
	assert.Equal(t, ErrNilPauseHandler, err)

	nftCreate, err = NewDCTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		nil,
	)
	assert.True(t, check.IfNil(nftCreate))
	assert.Equal(t, ErrNilRolesHandler, err)
}

func TestNewDCTNFTCreateFunc(t *testing.T) {
	t.Parallel()

	nftCreate, err := NewDCTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		&mock.DCTRoleHandlerStub{},
	)
	assert.False(t, check.IfNil(nftCreate))
	assert.Nil(t, err)
}

func TestDctNFTCreate_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	nftCreate := createNftCreateWithStubArguments()
	nftCreate.SetNewGasConfig(nil)
	assert.Equal(t, uint64(1), nftCreate.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, nftCreate.gasConfig)

	gasCost := createMockGasCost()
	nftCreate.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.DCTNFTCreate, nftCreate.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, nftCreate.gasConfig)
}

func TestDctNFTCreate_ProcessBuiltinFunctionInvalidArguments(t *testing.T) {
	t.Parallel()

	nftCreate := createNftCreateWithStubArguments()
	sender := mock.NewAccountWrapMock([]byte("address"))
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("caller"),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: []byte("recipient"),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilUserAccount, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNotEnoughGas, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  sender.AddressBytes(),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), []byte("arg2")},
			GasProvided: 1,
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.True(t, errors.Is(err, ErrInvalidArguments))
}

func TestDctNFTCreate_ProcessBuiltinFunctionNotAllowedToExecute(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("expected error")
	nftCreate, _ := NewDCTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		&mock.DCTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
				return expectedErr
			},
		},
	)
	sender := mock.NewAccountWrapMock([]byte("address"))
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  make([][]byte, 7),
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, expectedErr, err)
}

func TestDctNFTCreate_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	nftCreate, _ := NewDCTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.PauseHandlerStub{},
		&mock.DCTRoleHandlerStub{},
	)
	address := bytes.Repeat([]byte{1}, 32)
	sender := mock.NewUserAccount(address)
	//add some data in the trie, otherwise the creation will fail (it won't happen in real case usage as the create NFT
	//will be called after the creation permission was set in the account's data)
	_ = sender.AccountDataHandler().SaveKeyValue([]byte("key"), []byte("value"))

	token := "token"
	quantity := big.NewInt(2)
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attibutes := []byte("attributes")
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte(token),
				quantity.Bytes(),
				[]byte(name),
				big.NewInt(int64(royalties)).Bytes(),
				hash,
				attibutes,
				uris[0],
				uris[1],
			},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, err)
	require.NotNil(t, vmOutput)

	createdDct, latestNonce := readNFTData(t, sender, nftCreate.marshalizer, []byte(token), 1, address)
	assert.Equal(t, uint64(1), latestNonce)
	expectedDct := &dct.DCToken{
		Type:       uint32(vmcommon.NonFungible),
		Value:      quantity,
		Properties: nil,
		TokenMetaData: &dct.MetaData{
			Nonce:      1,
			Name:       []byte(name),
			Creator:    address,
			Royalties:  uint32(royalties),
			Hash:       hash,
			URIs:       uris,
			Attributes: attibutes,
		},
	}
	assert.Equal(t, expectedDct, createdDct)
}

func readNFTData(t *testing.T, account vmcommon.UserAccountHandler, marshalizer vmcommon.Marshalizer, tokenID []byte, nonce uint64, _ []byte) (*dct.DCToken, uint64) {
	nonceKey := getNonceKey(tokenID)
	latestNonceBytes, err := account.(vmcommon.UserAccountHandler).AccountDataHandler().RetrieveValue(nonceKey)
	require.Nil(t, err)
	latestNonce := big.NewInt(0).SetBytes(latestNonceBytes).Uint64()

	createdTokenID := []byte(vmcommon.DharitriProtectedKeyPrefix + vmcommon.DCTKeyIdentifier)
	createdTokenID = append(createdTokenID, tokenID...)
	tokenKey := computeDCTNFTTokenKey(createdTokenID, nonce)
	data, err := account.(vmcommon.UserAccountHandler).AccountDataHandler().RetrieveValue(tokenKey)
	require.Nil(t, err)

	dctData := &dct.DCToken{}
	err = marshalizer.Unmarshal(dctData, data)
	require.Nil(t, err)

	return dctData, latestNonce
}
