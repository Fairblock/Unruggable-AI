package broadcast

import (
	"context"
	"fmt"
	"io/ioutil"

	"cosmossdk.io/math"
	"google.golang.org/grpc"

	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	dcrdSecp256k1 "github.com/decred/dcrd/dcrec/secp256k1"
	"github.com/skip-mev/block-sdk/v2/testutils"
)


const (
	defaultGasAdjustment = 3
	defaultGasLimit      = 300000
)

type CosmosClient struct {
	authClient      authtypes.QueryClient
	txClient        tx.ServiceClient
	bankQueryClient banktypes.QueryClient

	GrpcConn   *grpc.ClientConn
	privateKey secp256k1.PrivKey
	publicKey  cryptotypes.PubKey
	account    authtypes.BaseAccount
	accAddress sdk.AccAddress
	chainID    string
	Dcrd_sk    *dcrdSecp256k1.PrivateKey
}

func LoadPublicKeysFromJSON(filePath string) ([]*dcrdSecp256k1.PublicKey, error) {

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	var base64Keys []string

	if err := json.Unmarshal(data, &base64Keys); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	var partyPKList []*dcrdSecp256k1.PublicKey

	for idx, b64Str := range base64Keys {

		keyBytes, err := base64.StdEncoding.DecodeString(b64Str)
		if err != nil {
			return nil, fmt.Errorf("error decoding Base64 string at index %d: %w", idx, err)
		}

		pk, err := dcrdSecp256k1.ParsePubKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing public key at index %d: %w", idx, err)
		}

		partyPKList = append(partyPKList, pk)
	}

	return partyPKList, nil
}
func LoadAddressesFromJSON(filePath string) ([]string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var addrStrings []string
	if err := json.Unmarshal(data, &addrStrings); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	return addrStrings, nil
}

func NewCosmosClient(endpoint, privKeyHex, chainID string) (*CosmosClient, error) {
	grpcConn, err := grpc.Dial(endpoint, grpc.WithInsecure(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(50*1024*1024), 
		grpc.MaxCallSendMsgSize(50*1024*1024),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC: %w", err)
	}

	authC := authtypes.NewQueryClient(grpcConn)
	bankC := banktypes.NewQueryClient(grpcConn)
	txC := tx.NewServiceClient(grpcConn)

	keyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode privkey hex: %w", err)
	}

	privKey := secp256k1.PrivKey{Key: keyBytes}
	pubKey := privKey.PubKey()

	dcrdPrivKey, _ := dcrdSecp256k1.PrivKeyFromBytes(keyBytes)

	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("fairy", "fairypub")
	cfg.SetBech32PrefixForValidator("fairyvaloper", "fairyvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("fairyvalcons", "fairyvalconspub")

	accAddr := sdk.AccAddress(pubKey.Address())

	resp, err := authC.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: accAddr.String(),
	})
	if err != nil {
		return nil, err
	}
	var baseAccount authtypes.BaseAccount
	if err := baseAccount.Unmarshal(resp.Account.Value); err != nil {
		return nil, err
	}

	return &CosmosClient{
		authClient:      authC,
		txClient:        txC,
		bankQueryClient: bankC,
		GrpcConn:        grpcConn,
		privateKey:      privKey,
		publicKey:       pubKey,
		account:         baseAccount,
		accAddress:      accAddr,
		chainID:         chainID,
		Dcrd_sk:         dcrdPrivKey,
	}, nil
}

func (c *CosmosClient) GetAddress() string {
	return c.account.Address
}

func (c *CosmosClient) updateAccSequence() error {
	out, err := c.authClient.Account(context.Background(),
		&authtypes.QueryAccountRequest{
			Address: c.accAddress.String(),
		})
	if err != nil {
		return err
	}
	var ba authtypes.BaseAccount
	if err := ba.Unmarshal(out.Account.Value); err != nil {
		return err
	}
	c.account = ba
	return nil
}

func (c *CosmosClient) BroadcastTx(msg sdk.Msg, adjustGas bool) (*tx.GetTxResponse, error) {
	if err := c.updateAccSequence(); err != nil {
		return nil, err
	}

	txBytes, err := c.signTxMsg(msg, adjustGas)
	if err != nil {
		return nil, err
	}

	resp, err := c.txClient.BroadcastTx(
		context.Background(),
		&tx.BroadcastTxRequest{
			TxBytes: txBytes,
			Mode:    tx.BroadcastMode_BROADCAST_MODE_SYNC,
		},
	)
	if err != nil {
		return nil, err
	}
	fmt.Println(resp)
	for {
		getTxResp, err := c.txClient.GetTx(context.Background(), &tx.GetTxRequest{Hash: resp.TxResponse.TxHash})
		if err != nil {

			if strings.Contains(err.Error(), "not found") {
				time.Sleep(time.Second)
				continue
			}
			return nil, err
		}

		return getTxResp, err
	}
}

func (c *CosmosClient) signTxMsg(msg sdk.Msg, adjustGas bool) ([]byte, error) {
	encodingCfg := testutils.CreateTestEncodingConfig()
	txBuilder := encodingCfg.TxConfig.NewTxBuilder()
	encodingCfg.TxConfig.SignModeHandler().DefaultMode()

	err := txBuilder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}

var newGasLimit uint64 = defaultGasLimit
	if adjustGas {
		txf := clienttx.Factory{}.
			WithGas(defaultGasLimit).
			WithSignMode(1).
			WithTxConfig(encodingCfg.TxConfig).
			WithChainID(c.chainID).
			WithAccountNumber(c.account.AccountNumber).
			WithSequence(c.account.Sequence).
			WithGasAdjustment(defaultGasAdjustment)

		_, newGasLimit, err = clienttx.CalculateGas(c.GrpcConn, txf, msg)
		if err != nil {
			return nil, err
		}
	}

	txBuilder.SetGasLimit(newGasLimit)
	feeAmount := sdk.NewCoins(sdk.NewCoin("ufairy", math.NewInt(800))) 
	txBuilder.SetFeeAmount(feeAmount)

	txBuilder.SetFeeAmount(feeAmount)
	signerData := authsigning.SignerData{
		ChainID:       c.chainID,
		AccountNumber: c.account.AccountNumber,
		Sequence:      c.account.Sequence,
		PubKey:        c.publicKey,
		Address:       c.account.Address,
	}

	sigData := signing.SingleSignatureData{
		SignMode:  1,
		Signature: nil,
	}
	sig := signing.SignatureV2{
		PubKey:   c.publicKey,
		Data:     &sigData,
		Sequence: c.account.Sequence,
	}

	if err := txBuilder.SetSignatures(sig); err != nil {
		return nil, err
	}

	sigV2, err := clienttx.SignWithPrivKey(
		context.Background(), 1, signerData, txBuilder, &c.privateKey,
		encodingCfg.TxConfig, c.account.Sequence,
	)

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, err
	}

	txBytes, err := encodingCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}




func (c *CosmosClient) QueryContractSmart(
    ctx context.Context,
    contractAddr string,
    msg []byte,
) ([]byte, error) {
   
    if c.GrpcConn == nil {
        return nil, fmt.Errorf("grpcConn not initialized")
    }

   
    queryClient := wasmtypes.NewQueryClient(c.GrpcConn)

   
    req := &wasmtypes.QuerySmartContractStateRequest{
        Address:   contractAddr,
        QueryData: msg,
    }

 
    resp, err := queryClient.SmartContractState(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("QuerySmartContractState error: %w", err)
    }

  
    return resp.Data, nil
}