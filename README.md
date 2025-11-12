# Unstoppable and Unruggable AI Agents

Read this blog and tutorial: https://phala.com/posts/phala-x-fairblock-a-blueprint-for-unruggable-ai-with-an-mpctee-hybrid-architecture
## TEE memory is transient
Confidential computing is rapidly expanding across domains such as Confidential AI, DeFi, and AI Agents. However, enclave memory is transient, meaning its contents, including cryptographic keys, are lost when the enclave shuts down. Without a secure mechanism to persist this information, critical data may become permanently inaccessible, putting funds and operations at risk.

## Multimodal confidential computing
A practical solution to this challenge lies in combining Multi-Party Computation (MPC) networks with decentralized storage systems. MPC networks distribute secrets across multiple nodes, ensuring no single node holds the complete key while allowing the network to reconstruct it when needed. Encrypted data can be securely stored on-chain, and when necessary, the MPC network can provide the key to a new enclave running the same image, provided specific conditions are met. This approach strengthens resilience and security, ensuring data accessibility and confidentiality even in untrusted environments.

[This article serves as the reference for more technical details on TEE vulnerabilities and the use of MPC.](https://www.bedlamresear.ch/posts/securing-tee-apps/#use-case-dependent-suggestions)

## Fairblock
This repository demonstrates a way to securely store and recover sensitive data, like private keys, from secure enclaves. If the TEE fails and the enclave data is lost, this setup ensures the key can still be retrieved.

Fairblock's dynamic MPC mechanism within FairyRing enables secure storage of encrypted data and private decryption once specific conditions are met. The Trusted Execution Environment (TEE) periodically encrypts and submits the private key to a smart contract. If the TEE ceases operation and submissions stop, the contract detects this and grants an authorized address the ability to request decryption and recover the key.
[Fairblock Private Decryption](https://docs.fairblock.network/docs/build/fairyring/fairyring_private_decryption)


## Phala
While this approach is compatible with any TEE (including local TEEs), we are thrilled to collaborate with Phala Network, leveraging the power of an AI agent within Phala Network's cutting-edge cloud infrastructure.
[Phala Network Cloud](cloud.phala.network)

# Other Applications of MPC+TEE  

While this work focuses on enabling unstoppable and unruggable confidential computing through MPC and TEEs, this multimodal approach can also be applied to other critical security scenarios, such as:  

- **Securing TEE Root Keys with MPC**  
  Using MPC to store the private root key of TEEs can help mitigate attacks similar to [recent incidents](https://www.theregister.com/2024/08/27/intel_root_key_xeons/).  

- **Enhancing Security by Combining TEE and MPC**  
  Integrating TEE and MPC ensures that neither depends solely on a single TEE’s hardware security nor on the honest majority assumption of MPCs.  

For more details, check out [this thread and repo](https://x.com/0xfairblock/status/1867585359896556026) on Fairblock’s work.


## Technical Overview

This repository consists of two main parts:

1. **CosmWasm Contract**
Located at `contract/src/contract.rs`, this contract manages private identity encryption and decryption. Every month, the encrypted private key is submitted to this contract. If no key is submitted for over a month, the contract assumes the TEE is no longer functioning and allows an authorized address to request decryption and recover the key.

2. **Cloud Code**
The cloud code (`cloud/main.go`) runs inside the TEE server. It reads the private key from a file, encrypts it using the public key from FairyRing, and submits the encrypted key to the contract every month.

For this example, the private key is assumed to be in a file at `cloud/key.txt`. Although the key is set to be submitted monthly by default, it can be adjusted in both the contract and the cloud code.

## Local Testing
### Clone Required Repositories
   
In order to test the example, clone FairyRing, FairyRingClient, and ShareGenerationClient outside this directory as follows:
```
cd ..

git clone https://github.com/Fairblock/fairyring.git
git clone git@github.com:Fairblock/fairyringclient.git
git clone git@github.com:Fairblock/sharegenerationclient.git

```
Make sure to switch to `contracts` branch of FairyRing.

### Set Up the Devnet and the Contract
   
Run the `setup.sh` script to start the FairyRing devnet, deploy the contract, and initialize it.

### Run the Cloud Code Locally
 
Once the chain is running and the contract is deployed, run the cloud code as follows:
```
cd cloud
go run main.go
```
This code needs configuration details like the chain endpoints, contract address, and the authorized address for private decryption. These should be set in a `.env` file. For testing, you can use the values provided in `example.env`. Note that for easier testing, the time period for submitting the key can be reduced in the contract and cloud code. 

At this point, the cloud code should automatically submit the encrypted key to the contract at the defined intervals.

To simulate a TEE failure and test the recovery process, first, stop the cloud code. Once the required time has passed since the last key submission, run the `recovery.sh` script. This will request a private decryption, retrieve the decryption key, and perform the decryption to recover the private key. 

## Cloud Testing

In order to test the code with Phala Cloud, use the provided dockerfile located in `cloud/dockerfile` to generate the corresponding image for the cloud code. Before building the Docker image, make sure that the config values in the `.env` file are correctly set. The rest of the process will be similar to local testing.
In a real-world scenario, the actual private key should be saved in cloud/key.txt so the cloud code can access it.
