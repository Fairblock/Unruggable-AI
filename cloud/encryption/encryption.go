package encryption

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"

	bls "github.com/drand/kyber-bls12381"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	enc "github.com/FairBlock/DistributedIBE/encryption"
	"cloud_encryption/broadcast"
)

// DoEncryptionAndSubmit encrypts the plaintext from keyFile and submits the encrypted data.
func DoEncryptionAndSubmit(client *broadcast.CosmosClient, identity, contractAddress, keyFile, newPubKey string) error {
	plainBytes, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("read key file %s: %w", keyFile, err)
	}

	suite := bls.NewBLS12381Suite()
	publicKeyBytes, err := hex.DecodeString(newPubKey)
	if err != nil {
		return fmt.Errorf("decode public key hex: %w", err)
	}
	publicKeyPoint := suite.G1().Point()
	if err := publicKeyPoint.UnmarshalBinary(publicKeyBytes); err != nil {
		return fmt.Errorf("unmarshal BLS pubkey: %w", err)
	}

	var destCipherData bytes.Buffer
	var plainTextBuffer bytes.Buffer
	if _, err := plainTextBuffer.Write(plainBytes); err != nil {
		return fmt.Errorf("write plaintext: %w", err)
	}
	if err := enc.Encrypt(publicKeyPoint, []byte(identity), &destCipherData, &plainTextBuffer); err != nil {
		return fmt.Errorf("encrypt data: %w", err)
	}
	hexCipher := hex.EncodeToString(destCipherData.Bytes())

	executeMsg := map[string]interface{}{
		"store_encrypted_data": map[string]interface{}{
			"identity": identity,
			"data":     hexCipher,
		},
	}
	msgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return fmt.Errorf("marshal store_encrypted_data msg: %w", err)
	}
	wasmMsg := &wasmtypes.MsgExecuteContract{
		Sender:   client.GetAddress(),
		Contract: contractAddress,
		Msg:      msgBytes,
		Funds:    nil,
	}
	txResp, err := client.BroadcastTx(wasmMsg, true)
	if err != nil {
		return fmt.Errorf("broadcast store_encrypted_data error: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("store_encrypted_data failed: code=%d, raw_log=%s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	return nil
}
