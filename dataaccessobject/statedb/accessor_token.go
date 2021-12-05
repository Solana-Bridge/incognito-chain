package statedb

import (
	"encoding/json"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject"
)

func StorePrivacyToken(stateDB *StateDB, tokenID common.Hash, name string, symbol string, tokenType int, mintable bool, amount uint64, info []byte, txHash common.Hash) (uint64, error) {
	dataaccessobject.Logger.Log.Infof("Store Privacy Token %+v, txHash %+v\n", tokenID, txHash.String())
	key := GenerateTokenObjectKey(tokenID)
	_, has, err := stateDB.getTokenState(key)
	if err != nil {
		return 0, NewStatedbError(StorePrivacyTokenError, err)
	}
	if has {
		dataaccessobject.Logger.Log.Infof("Token %v already existed\n", tokenID.String())
		return 0, nil
	}
	value := NewTokenStateWithValue(tokenID, name, symbol, tokenType, mintable, amount, info, txHash)
	err = stateDB.SetStateObject(TokenObjectType, key, value)
	if err != nil {
		return 0, NewStatedbError(StorePrivacyTokenError, err)
	}
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return 0, NewStatedbError(StorePrivacyTokenError, err)
	}
	return uint64(len(valueBytes)), nil
}

func StorePrivacyTokenTx(stateDB *StateDB, tokenID common.Hash, txHash common.Hash) (uint64, error) {
	keyToken := GenerateTokenObjectKey(tokenID)
	_, has, err := stateDB.getTokenState(keyToken)
	if err != nil {
		return 0, NewStatedbError(GetPrivacyTokenError, err)
	}
	if !has {
		_, err := StorePrivacyToken(stateDB, tokenID, "", "", UnknownToken, false, 0, []byte{}, txHash)
		if err != nil {
			return 0, err
		}
	}
	keyTokenTx := GenerateTokenTransactionObjectKey(tokenID, txHash)
	tokenTransactionState := NewTokenTransactionStateWithValue(txHash)
	valueBytes, err := json.Marshal(tokenTransactionState)
	if err != nil {
		return 0, NewStatedbError(StorePrivacyTokenTransactionError, err)
	}
	err = stateDB.SetStateObject(TokenTransactionObjectType, keyTokenTx, tokenTransactionState)
	if err != nil {
		return 0, NewStatedbError(StorePrivacyTokenTransactionError, err)
	}
	return uint64(len(valueBytes)), nil
}

func ListPrivacyToken(stateDB *StateDB) map[common.Hash]*TokenState {
	return stateDB.getAllToken()
}

func ListPrivacyTokenWithTxs(stateDB *StateDB) map[common.Hash]*TokenState {
	return stateDB.getAllTokenWithTxs()
}

func GetPrivacyTokenTxs(stateDB *StateDB, tokenID common.Hash) []common.Hash {
	txs := stateDB.getTokenTxs(tokenID)
	return txs
}

func PrivacyTokenIDExisted(stateDB *StateDB, tokenID common.Hash) bool {
	key := GenerateTokenObjectKey(tokenID)
	tokenState, has, err := stateDB.getTokenState(key)
	if err != nil {
		return false
	}
	tempTokenID := tokenState.TokenID()
	if has && !tempTokenID.IsEqual(&tokenID) {
		panic("same key wrong value")
	}
	return has
}

func GetPrivacyTokenState(stateDB *StateDB, tokenID common.Hash) (*TokenState, bool, error) {
	key := GenerateTokenObjectKey(tokenID)
	tokenState, has, err := stateDB.getTokenState(key)
	if err != nil {
		return nil, false, err
	}
	tempTokenID := tokenState.TokenID()
	if has && !tempTokenID.IsEqual(&tokenID) {
		panic("same key wrong value")
	}
	if !has {
		return tokenState, false, nil
	}
	txs := GetPrivacyTokenTxs(stateDB, tokenID)
	tokenState.AddTxs(txs)
	return tokenState, true, nil
}
