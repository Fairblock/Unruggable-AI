# Phala Network Key Recovery

This repository provides an example of how essential secret data such as private keys inside enclaves can be securely and confidentially stored in untrusted environments to be recoverable in case the TEE stops working and the data inside the enclave is lost. 
For this purpose, the private identity encryption mechanism in FairyRing is used which allows for storing encrypted data and private decryption of it once a specific condition is met. 

## Overview

This repository consists of to main parts:

1. A CosmWasm contract (`contract/src/contract.rs`) which manages the private-identity encryption and decryption. The encrypted private key will be submitted to this contract every month. In case the encrypted key is not submitted for over a month, the contract assumes that the TEE has stopped working and allows for the key recovery through submitting private decryption request by an authorized address.
2. The Go code (`cloud/main.go`) which will run inside the TEE server which is responsible for reading the private key, encrypting it with the correct public key fetched from FairyRing and submitting it on the contract every month.

For this example, the assumption is that the private key is stored in a file (`cloud/key.txt`) from which it can be read and encrypted by the cloud code. Moreover, the encrypted key submission period has been hardcoded as one month but it can be changed inside the contract and cloud code. 


## Local Testing

In order to test the example, clone FairyRing, FairyRingClient, and ShareGenerationClient outside this directory as follows:
```
cd ..

git clone https://github.com/Fairblock/fairyring.git
git clone git@github.com:Fairblock/fairyringclient.git
git clone git@github.com:Fairblock/sharegenerationclient.git

```
Next, run the `setup.sh` script to run fairyring devnet, and deploy and initialize the contract.
Once the chain is ready and the contract is deployed, run the `cloud/main.go`:
```
cd cloud
go run main.go
```
This code requires the chain endpoints, contract address, the authorized address for private decryption etc. to be set in a `.env` file. For a local test, the provided `example.env` values can be used. Note, that for easier testing, the time period for submitting the key can be reduced in  contract and cloud code. 

At this stage, the Go code should be repeatedly submitting the encrypted key to the contract based on the defined time period. 

