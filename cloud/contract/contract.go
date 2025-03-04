package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"cloud_encryption/broadcast"
	"cloud_encryption/utils"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	peptypes "github.com/Fairblock/fairyring/x/pep/types"
)

type QueryResponse struct {
	ActivePubKey PubKeyInfo `json:"active_pubkey"`
	QueuedPubKey PubKeyInfo `json:"queued_pubkey"`
}

type PubKeyInfo struct {
	PublicKey string `json:"public_key"`
	Creator   string `json:"creator"`
	Expiry    string `json:"expiry"`
}

func RequestNewIdentity(client *broadcast.CosmosClient, contractAddress, authorizedAddr string) (string, error) {
	executeMsg := map[string]interface{}{
		"request_identity": map[string]interface{}{
			"authorized_address": authorizedAddr,
		},
	}
	msgBytes, err := json.Marshal(executeMsg)
	if err != nil {
		return "", fmt.Errorf("marshal executeMsg: %w", err)
	}
	wasmMsg := &wasmtypes.MsgExecuteContract{
		Sender:   client.GetAddress(),
		Contract: contractAddress,
		Msg:      msgBytes,
		Funds:    nil,
	}
	txResp, err := client.BroadcastTx(wasmMsg, true)
	if err != nil {
		return "", err
	}
	if txResp.TxResponse.Code != 0 {
		return "", fmt.Errorf("tx failed: code=%d, raw_log=%s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	return "pending-identity", nil
}

func FetchNewlyCreatedIdentity(client *broadcast.CosmosClient, contractAddress, authorizedAddr string) (string, error) {
	queryBody := map[string]interface{}{
		"get_all_identity": struct{}{},
	}
	queryBytes, err := json.Marshal(queryBody)
	if err != nil {
		return "", fmt.Errorf("marshal query msg: %w", err)
	}
	rawResp, err := client.QueryContractSmart(context.Background(), contractAddress, queryBytes)
	if err != nil {
		return "", fmt.Errorf("smart query error: %w", err)
	}
	var response struct {
		Records []struct {
			Identity string `json:"identity"`
			Creator  string `json:"creator"`
		} `json:"records"`
	}
	if err := json.Unmarshal(rawResp, &response); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	var foundIdentity string
	for _, rec := range response.Records {
		if rec.Creator == authorizedAddr {
			foundIdentity = rec.Identity
		}
	}
	if foundIdentity == "" {
		return "", fmt.Errorf("no identity found for authorizedAddr=%s", authorizedAddr)
	}
	return foundIdentity, nil
}

func RegisterContractOnFairyring(client *broadcast.CosmosClient, contractAddr, uniqueID string) error {
	msg := &peptypes.MsgRegisterContract{
		Creator:         client.GetAddress(),
		ContractAddress: contractAddr,
		Identity:        uniqueID,
	}
	txResp, err := client.BroadcastTx(msg, true)
	if err != nil {
		return fmt.Errorf("broadcast register-contract error: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("register-contract failed: code=%d, log=%s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	return nil
}

func UpdateContractPubKey(client *broadcast.CosmosClient, contractAddress, newPubKey string) error {
	updateMsg := map[string]interface{}{
		"update_pubkey": map[string]interface{}{
			"pubkey": newPubKey,
		},
	}
	msgBytes, err := json.Marshal(updateMsg)
	if err != nil {
		return fmt.Errorf("marshal update_pubkey msg: %w", err)
	}
	wasmMsg := &wasmtypes.MsgExecuteContract{
		Sender:   client.GetAddress(),
		Contract: contractAddress,
		Msg:      msgBytes,
		Funds:    nil,
	}
	txResp, err := client.BroadcastTx(wasmMsg, true)
	if err != nil {
		return fmt.Errorf("broadcast update_pubkey error: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("update_pubkey failed: code=%d, raw_log=%s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	return nil
}


func FetchPublicKey() (string, error) {
	baseURL := utils.GetEnv("FAIRYRING_REST_URL")
	url := baseURL + "/fairyring/keyshare/pubkey"

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("request public key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}
	var pkResp QueryResponse
	if err := json.Unmarshal(respBytes, &pkResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	return pkResp.QueuedPubKey.PublicKey, nil
}
