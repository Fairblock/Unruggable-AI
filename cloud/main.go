package main

import (
	"fmt"
	"log"
	"time"
	"github.com/joho/godotenv"
	"cloud_encryption/contract"
	"cloud_encryption/encryption"
	"cloud_encryption/utils"
	"cloud_encryption/broadcast"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Failed to load .env file: %v", err)
	}

	// Read configuration from .env.
	rpcEndpoint := utils.GetEnv("COSMOS_GRPC_ENDPOINT")
	privateKeyHex := utils.GetEnv("COSMOS_PRIVATE_KEY_HEX")
	chainID := utils.GetEnv("COSMOS_CHAIN_ID")
	contractAddress := utils.GetEnv("CONTRACT_ADDRESS")
	authorizedAddr := utils.GetEnv("AUTHORIZED_ADDRESS")
	keyFile := utils.GetEnv("PLAINTEXT_FILE")

	client, err := broadcast.NewCosmosClient(rpcEndpoint, privateKeyHex, chainID)
	if err != nil {
		log.Fatalf("Cosmos client creation failed: %v", err)
	}

	// Request a new identity.
	if _, err := contract.RequestNewIdentity(client, contractAddress, authorizedAddr); err != nil {
		log.Fatalf("Identity request failed: %v", err)
	}
	fmt.Println("Identity requested.")

	// Allow time for identity creation.
	time.Sleep(5 * time.Second)

	identity, err := contract.FetchNewlyCreatedIdentity(client, contractAddress, authorizedAddr)
	if err != nil {
		log.Fatalf("Fetching identity failed: %v", err)
	}
	fmt.Printf("New identity: %s\n", identity)

	if err := contract.RegisterContractOnFairyring(client, contractAddress, identity); err != nil {
		log.Fatalf("Contract registration failed: %v", err)
	}
	fmt.Printf("Contract %s registered with identity %s.\n", contractAddress, identity)

	// Main loop: update public key and submit encrypted data.
	for {
		newPk, err := contract.FetchPublicKey()
		if err != nil {
			log.Printf("Fetching public key failed: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Println(newPk)

		if err := contract.UpdateContractPubKey(client, contractAddress, newPk); err != nil {
			log.Printf("Updating public key failed: %v", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if err := encryption.DoEncryptionAndSubmit(client, identity, contractAddress, keyFile, newPk); err != nil {
			log.Printf("Encryption and submission failed: %v", err)
		} else {
			log.Printf("Stored new encrypted key for identity %s", identity)
		}

		time.Sleep(20 * time.Second)
	}
}
