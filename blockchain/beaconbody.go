package blockchain

import (
	"encoding/json"

	"github.com/ninjadotorg/constant/common"
)

type BeaconBlockBody struct {
	ShardState   [][]common.Hash
	Instructions [][]string
}

func (self *BeaconBlockBody) toString() string {
	res := ""

	for _, l := range self.ShardState {
		for _, r := range l {
			res += r.String()
		}
	}

	for _, l := range self.Instructions {
		for _, r := range l {
			res += r
		}
	}

	return res
}

func (self *BeaconBlockBody) Hash() common.Hash {
	return common.DoubleHashH([]byte(self.toString()))
}

func (self *BeaconBlockBody) UnmarshalJSON(data []byte) error {
	blkBody := &BeaconBlockBody{}

	err := json.Unmarshal(data, blkBody)
	if err != nil {
		return NewBlockChainError(UnmashallJsonBlockError, err)
	}
	self = blkBody
	return nil
}
