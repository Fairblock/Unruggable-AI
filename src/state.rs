use std::collections::HashMap;
use crate::msg::IndexedEncryptedKeyshare;
use cw_storage_plus::{Item, Map};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct IdentityRecord {
    pub identity: String,
    pub pubkey: String,
    pub creator: String,
    pub encrypted_data: String,
    pub last_submission: u64,
    pub private_keyshares: HashMap<String, Vec<IndexedEncryptedKeyshare>>,
}

pub const RECORDS: Map<&str, IdentityRecord> = Map::new("records");

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct PendingRequest {
    pub creator: String,
    pub authorized_address: String,
}

pub const PENDING_REQUESTS: Map<u64, PendingRequest> = Map::new("pending_requests");
pub const LAST_REPLY_ID: Item<u64> = Item::new("last_reply_id");
pub const PUBKEY: Item<String> = Item::new("pubkey");
pub const REQUESTER: Item<String> = Item::new("requester");
