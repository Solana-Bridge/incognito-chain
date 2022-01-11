// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	incognitokey "github.com/incognitochain/incognito-chain/incognitokey"
	coin "github.com/incognitochain/incognito-chain/privacy/coin"

	key "github.com/incognitochain/incognito-chain/privacy/key"

	mock "github.com/stretchr/testify/mock"

	operation "github.com/incognitochain/incognito-chain/privacy/operation"
)

// Coin is an autogenerated mock type for the Coin type
type Coin struct {
	mock.Mock
}

// Bytes provides a mock function with given fields:
func (_m *Coin) Bytes() []byte {
	ret := _m.Called()

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0
}

// CheckCoinValid provides a mock function with given fields: _a0, _a1, _a2
func (_m *Coin) CheckCoinValid(_a0 key.PaymentAddress, _a1 []byte, _a2 uint64) bool {
	ret := _m.Called(_a0, _a1, _a2)

	var r0 bool
	if rf, ok := ret.Get(0).(func(key.PaymentAddress, []byte, uint64) bool); ok {
		r0 = rf(_a0, _a1, _a2)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Decrypt provides a mock function with given fields: _a0
func (_m *Coin) Decrypt(_a0 *incognitokey.KeySet) (coin.PlainCoin, error) {
	ret := _m.Called(_a0)

	var r0 coin.PlainCoin
	if rf, ok := ret.Get(0).(func(*incognitokey.KeySet) coin.PlainCoin); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(coin.PlainCoin)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*incognitokey.KeySet) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DoesCoinBelongToKeySet provides a mock function with given fields: keySet
func (_m *Coin) DoesCoinBelongToKeySet(keySet *incognitokey.KeySet) (bool, *operation.Point) {
	ret := _m.Called(keySet)

	var r0 bool
	if rf, ok := ret.Get(0).(func(*incognitokey.KeySet) bool); ok {
		r0 = rf(keySet)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 *operation.Point
	if rf, ok := ret.Get(1).(func(*incognitokey.KeySet) *operation.Point); ok {
		r1 = rf(keySet)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*operation.Point)
		}
	}

	return r0, r1
}

// GetAssetTag provides a mock function with given fields:
func (_m *Coin) GetAssetTag() *operation.Point {
	ret := _m.Called()

	var r0 *operation.Point
	if rf, ok := ret.Get(0).(func() *operation.Point); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operation.Point)
		}
	}

	return r0
}

// GetCoinDetailEncrypted provides a mock function with given fields:
func (_m *Coin) GetCoinDetailEncrypted() []byte {
	ret := _m.Called()

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0
}

// GetCoinID provides a mock function with given fields:
func (_m *Coin) GetCoinID() [32]byte {
	ret := _m.Called()

	var r0 [32]byte
	if rf, ok := ret.Get(0).(func() [32]byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([32]byte)
		}
	}

	return r0
}

// GetCommitment provides a mock function with given fields:
func (_m *Coin) GetCommitment() *operation.Point {
	ret := _m.Called()

	var r0 *operation.Point
	if rf, ok := ret.Get(0).(func() *operation.Point); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operation.Point)
		}
	}

	return r0
}

// GetInfo provides a mock function with given fields:
func (_m *Coin) GetInfo() []byte {
	ret := _m.Called()

	var r0 []byte
	if rf, ok := ret.Get(0).(func() []byte); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	return r0
}

// GetKeyImage provides a mock function with given fields:
func (_m *Coin) GetKeyImage() *operation.Point {
	ret := _m.Called()

	var r0 *operation.Point
	if rf, ok := ret.Get(0).(func() *operation.Point); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operation.Point)
		}
	}

	return r0
}

// GetPublicKey provides a mock function with given fields:
func (_m *Coin) GetPublicKey() *operation.Point {
	ret := _m.Called()

	var r0 *operation.Point
	if rf, ok := ret.Get(0).(func() *operation.Point); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operation.Point)
		}
	}

	return r0
}

// GetRandomness provides a mock function with given fields:
func (_m *Coin) GetRandomness() *operation.Scalar {
	ret := _m.Called()

	var r0 *operation.Scalar
	if rf, ok := ret.Get(0).(func() *operation.Scalar); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operation.Scalar)
		}
	}

	return r0
}

// GetSNDerivator provides a mock function with given fields:
func (_m *Coin) GetSNDerivator() *operation.Scalar {
	ret := _m.Called()

	var r0 *operation.Scalar
	if rf, ok := ret.Get(0).(func() *operation.Scalar); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operation.Scalar)
		}
	}

	return r0
}

// GetShardID provides a mock function with given fields:
func (_m *Coin) GetShardID() (uint8, error) {
	ret := _m.Called()

	var r0 uint8
	if rf, ok := ret.Get(0).(func() uint8); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint8)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSharedConcealRandom provides a mock function with given fields:
func (_m *Coin) GetSharedConcealRandom() *operation.Scalar {
	ret := _m.Called()

	var r0 *operation.Scalar
	if rf, ok := ret.Get(0).(func() *operation.Scalar); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operation.Scalar)
		}
	}

	return r0
}

// GetSharedRandom provides a mock function with given fields:
func (_m *Coin) GetSharedRandom() *operation.Scalar {
	ret := _m.Called()

	var r0 *operation.Scalar
	if rf, ok := ret.Get(0).(func() *operation.Scalar); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*operation.Scalar)
		}
	}

	return r0
}

// GetTxRandom provides a mock function with given fields:
func (_m *Coin) GetTxRandom() *coin.TxRandom {
	ret := _m.Called()

	var r0 *coin.TxRandom
	if rf, ok := ret.Get(0).(func() *coin.TxRandom); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*coin.TxRandom)
		}
	}

	return r0
}

// GetValue provides a mock function with given fields:
func (_m *Coin) GetValue() uint64 {
	ret := _m.Called()

	var r0 uint64
	if rf, ok := ret.Get(0).(func() uint64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint64)
	}

	return r0
}

// GetVersion provides a mock function with given fields:
func (_m *Coin) GetVersion() uint8 {
	ret := _m.Called()

	var r0 uint8
	if rf, ok := ret.Get(0).(func() uint8); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(uint8)
	}

	return r0
}

// IsEncrypted provides a mock function with given fields:
func (_m *Coin) IsEncrypted() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// SetBytes provides a mock function with given fields: _a0
func (_m *Coin) SetBytes(_a0 []byte) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func([]byte) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
