RPC=http://localhost:26657
CHAIN_ID=fairyring_devnet
UNIQUE_ID="fairy1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq89ptu2/req-fairy1m9l358xunhhwds0568za49mzhvuxx9uxdra8sq-1"
PUB_KEY_64="A/MdHVpitzHNSdD1Zw3kY+L5PEIPyd9l6sD5i4aIfXp9"
CONTRACT_ADDRESS="fairy1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq89ptu2"
ACC_ADDR_W1="fairy1m9l358xunhhwds0568za49mzhvuxx9uxdra8sq"
PRIV_KEY_HEX_W1="a267fb03b3e6dc381550ea0257ace31433698f16248ba111dfb75550364d31fe"


fairyringd tx wasm execute $CONTRACT_ADDRESS '{"request_private_keyshare": {"identity": "'$UNIQUE_ID'", "secp_pubkey": "'$PUB_KEY_64'"}}' \
    --amount 400000ufairy --from wallet1 --gas 9000000 --chain-id fairyring_devnet -y


echo "Waiting for encrypted keyshare value to be available..."
for i in {1..10}; do
   JSON_OUTPUT=$(fairyringd q wasm contract-state smart "$CONTRACT_ADDRESS" '{"get_all_identity": {}}' --chain-id fairyring_devnet --node http://localhost:26657 | yq eval -o=json -N)


    ENCRYPTED_KEYSHARE_VALUE=$(echo "$JSON_OUTPUT" |yq eval -o=json |  jq -r '.data.records | last | .private_keyshares["'"$ACC_ADDR_W1"'"] | if . then .[0].encrypted_keyshare_value else empty end')

    FAIRYRING_PUBKEY=$(echo "$JSON_OUTPUT" |yq eval -o=json |  jq -r '.data.records | last | .pubkey')

    ENCRYPTED_DATA=$(echo "$JSON_OUTPUT" |yq eval -o=json |  jq -r '.data.records | last | .encrypted_data')

    if [[ -n "$ENCRYPTED_KEYSHARE_VALUE" && "$ENCRYPTED_KEYSHARE_VALUE" != "null" ]]; then
        echo "Retrieved ENCRYPTED_KEYSHARE_VALUE: $ENCRYPTED_KEYSHARE_VALUE"
        break
    fi

    echo "Waiting for keyshare availability... ($i/10)"
    sleep 3
done

# If still empty after retries, exit
if [[ -z "$ENCRYPTED_KEYSHARE_VALUE" || "$ENCRYPTED_KEYSHARE_VALUE" == "null" ]]; then
    echo "Error: Could not retrieve encrypted keyshare value. Exiting..."
    exit 1
fi


echo "Aggregating keyshares to obtain decryption key..."
DECRYPTION_KEY=$(fairyringd aggregate-keyshares "[ { \"encrypted_keyshare_value\": \"$ENCRYPTED_KEYSHARE_VALUE\", \"encrypted_keyshare_index\": 1 } ]" '""' $ACC_ADDR_W1 $PRIV_KEY_HEX_W1)

if [[ -z "$DECRYPTION_KEY" || "$DECRYPTION_KEY" == "null" ]]; then
    echo "Error: Could not retrieve decryption key. Exiting..."
    exit 1
fi
echo "Retrieved DECRYPTION_KEY: $DECRYPTION_KEY"

FAIRYRING_PUBKEY=$(fairyringd query pep show-active-pub-key -o json | jq -r '.active_pubkey.public_key')

echo "Decrypting final transaction data..."
DECRYPTED_MESSAGE=$(fairyringd query pep decrypt-data $FAIRYRING_PUBKEY $DECRYPTION_KEY $ENCRYPTED_DATA -o json | jq -r '.decrypted_data')

echo "Decrypted Message: $DECRYPTED_MESSAGE"