package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/Dharitri-org/me-vm-common"
	"github.com/Dharitri-org/me-vm-common/data/dct"
	"github.com/Dharitri-org/me-vm-common/mock"
	"github.com/stretchr/testify/require"
)

func TestNewDCTLocalMintFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		argsFunc func() (c uint64, m vmcommon.Marshalizer, p vmcommon.DCTPauseHandler, r vmcommon.DCTRoleHandler)
		exError  error
	}{
		{
			name: "NilMarshalizer",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.DCTPauseHandler, r vmcommon.DCTRoleHandler) {
				return 0, nil, &mock.PauseHandlerStub{}, &mock.DCTRoleHandlerStub{}
			},
			exError: ErrNilMarshalizer,
		},
		{
			name: "NilPauseHandler",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.DCTPauseHandler, r vmcommon.DCTRoleHandler) {
				return 0, &mock.MarshalizerMock{}, nil, &mock.DCTRoleHandlerStub{}
			},
			exError: ErrNilPauseHandler,
		},
		{
			name: "NilRolesHandler",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.DCTPauseHandler, r vmcommon.DCTRoleHandler) {
				return 0, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, nil
			},
			exError: ErrNilRolesHandler,
		},
		{
			name: "Ok",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.DCTPauseHandler, r vmcommon.DCTRoleHandler) {
				return 0, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.DCTRoleHandlerStub{}
			},
			exError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDCTLocalMintFunc(tt.argsFunc())
			require.Equal(t, err, tt.exError)
		})
	}
}

func TestDctLocalMint_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	dctLocalMintF, _ := NewDCTLocalMintFunc(0, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.DCTRoleHandlerStub{})

	dctLocalMintF.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{
		DCTLocalMint: 500},
	})

	require.Equal(t, uint64(500), dctLocalMintF.funcGasCost)
}

func TestDctLocalMint_ProcessBuiltinFunction_CalledWithValueShouldErr(t *testing.T) {
	t.Parallel()

	dctLocalMintF, _ := NewDCTLocalMintFunc(0, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.DCTRoleHandlerStub{})

	_, err := dctLocalMintF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(1),
		},
	})
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
}

func TestDctLocalMint_ProcessBuiltinFunction_CheckAllowToExecuteShouldErr(t *testing.T) {
	t.Parallel()

	localErr := errors.New("local err")
	dctLocalMintF, _ := NewDCTLocalMintFunc(0, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.DCTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return localErr
		},
	})

	_, err := dctLocalMintF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	})
	require.Equal(t, localErr, err)
}

func TestDctLocalMint_ProcessBuiltinFunction_CannotAddToDctBalanceShouldErr(t *testing.T) {
	t.Parallel()

	dctLocalMintF, _ := NewDCTLocalMintFunc(0, &mock.MarshalizerMock{}, &mock.PauseHandlerStub{}, &mock.DCTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return nil
		},
	})

	localErr := errors.New("local err")
	_, err := dctLocalMintF.ProcessBuiltinFunction(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					return nil, localErr
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					return localErr
				},
			}
		},
	}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
		},
	})
	require.Equal(t, localErr, err)
}

func TestDctLocalMint_ProcessBuiltinFunction_ShouldWork(t *testing.T) {
	t.Parallel()

	marshalizer := &mock.MarshalizerMock{}
	dctLocalMintF, _ := NewDCTLocalMintFunc(50, marshalizer, &mock.PauseHandlerStub{}, &mock.DCTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return nil
		},
	})

	sndAccout := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, error) {
					dctData := &dct.DCToken{Value: big.NewInt(100)}
					return marshalizer.Marshal(dctData)
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					dctData := &dct.DCToken{}
					_ = marshalizer.Unmarshal(dctData, value)
					require.Equal(t, big.NewInt(101), dctData.Value)
					return nil
				},
			}
		},
	}
	vmOutput, err := dctLocalMintF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, nil, err)

	expectedVMOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: 450,
		Logs: []*vmcommon.LogEntry{
			{
				Identifier: []byte("DCTLocalMint"),
				Address:    nil,
				Topics:     [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
				Data:       nil,
			},
		},
	}
	require.Equal(t, expectedVMOutput, vmOutput)

	mintTooMuch := make([]byte, 101)
	mintTooMuch[0] = 1
	vmOutput, err = dctLocalMintF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), mintTooMuch},
			GasProvided: 500,
		},
	})
	require.True(t, errors.Is(err, ErrInvalidArguments))
	require.Nil(t, vmOutput)
}
