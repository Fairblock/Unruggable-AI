RPC=http://localhost:26657
CHAIN_ID=fairyring_devnet

cd ../fairyring
make devnet-down
make devnet-up

cd ../test/contract

RUSTFLAGS='-C link-arg=-s -C target-feature=+bulk-memory' \
cargo build --release --target wasm32-unknown-unknown
docker run --rm -v "$(pwd)":/code   --mount type=volume,source="$(basename "$(pwd)")_cache",target=/code/target   --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry   cosmwasm/rust-optimizer:0.16.0

fairyringd tx wasm store artifacts/contract.wasm   --from wallet1   --chain-id $CHAIN_ID   --node $RPC  --gas auto   --gas-adjustment 1.3   --fees 2368ufairy   -y 
sleep 2

fairyringd tx wasm instantiate 2 '{"pubkey": ""}'   --from wallet1 --label "My CosmWasm Contract"  --chain-id $CHAIN_ID   --node $RPC  --no-admin   --gas auto --gas-adjustment 1.3   --fees 214ufairy   -y




