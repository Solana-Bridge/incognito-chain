package wire

import (
	"encoding/json"

	peer "github.com/libp2p/go-libp2p-peer"
)

type MessageAddr struct {
	RawAddresses []string
}

func (self MessageAddr) MessageType() string {
	return CmdAddr
}

func (self MessageAddr) MaxPayloadLength(pver int) int {
	return MaxBlockPayload
}

func (self MessageAddr) JsonSerialize() ([]byte, error) {
	jsonBytes, err := json.Marshal(self)
	return jsonBytes, err
}

func (self MessageAddr) JsonDeserialize(jsonStr string) error {
	err := json.Unmarshal([]byte(jsonStr), self)
	return err
}

func (self MessageAddr) SetSenderID(senderID peer.ID) error {
	return nil
}
