use cosmwasm_std::{CustomMsg, CustomQuery};
use schemars::JsonSchema;
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use serde::ser::SerializeStruct;
use fairblock_proto::fairyring::pep::MsgRequestPrivateIdentity;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub pubkey: String,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    UpdatePubkey { pubkey: String },
    RequestPrivateKeyshare { identity: String, secp_pubkey: String },
    RequestIdentity { authorized_address: String },
    StoreEncryptedData { identity: String, data: String },
    ExecuteContractPrivateMsg { identity: String, private_decryption_key: PrivateDecryptionKey },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct PrivateDecryptionKey {
    pub requester: String,
    pub private_keyshares: Vec<IndexedEncryptedKeyshare>,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub struct IndexedEncryptedKeyshare {
    pub encrypted_keyshare_value: String,
    pub encrypted_keyshare_index: i64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    GetIdentity { identity: String },
    GetAllIdentity {},
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum PepQuery {
    QueryPubkeyRequest {},
}

impl CustomQuery for PepQuery {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct IdentityResponse {
    pub record: crate::state::IdentityRecord,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct AllIdentitiesResponse {
    pub records: Vec<crate::state::IdentityRecord>,
}

#[derive(Debug, Clone, PartialEq)]
pub struct PepRequestMsg {
    pub inner: MsgRequestPrivateIdentity,
}

impl CustomMsg for PepRequestMsg {}

impl JsonSchema for PepRequestMsg {
    fn schema_name() -> String {
        "PepRequestMsg".to_string()
    }
    fn json_schema(_gen: &mut schemars::gen::SchemaGenerator) -> schemars::schema::Schema {
        use schemars::schema::{Schema, SchemaObject, InstanceType, ObjectValidation};
        Schema::Object(SchemaObject {
            instance_type: Some(InstanceType::Object.into()),
            object: Some(Box::new(ObjectValidation {
                properties: [
                    (
                        "creator".to_string(),
                        Schema::Object(SchemaObject {
                            instance_type: Some(InstanceType::String.into()),
                            ..Default::default()
                        }),
                    ),
                    (
                        "req_id".to_string(),
                        Schema::Object(SchemaObject {
                            instance_type: Some(InstanceType::String.into()),
                            ..Default::default()
                        }),
                    ),
                ]
                .iter()
                .cloned()
                .collect(),
                required: {
                    let mut req = std::collections::BTreeSet::new();
                    req.insert("creator".to_string());
                    req.insert("req_id".to_string());
                    req
                },
                ..Default::default()
            })),
            ..Default::default()
        })
    }
}

impl Serialize for PepRequestMsg {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        let mut state = serializer.serialize_struct("PepRequestMsg", 2)?;
        state.serialize_field("creator", &self.inner.creator)?;
        state.serialize_field("req_id", &self.inner.req_id)?;
        state.end()
    }
}

impl<'de> Deserialize<'de> for PepRequestMsg {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        #[derive(Deserialize)]
        struct Helper {
            creator: String,
            req_id: String,
        }
        let helper = Helper::deserialize(deserializer)?;
        Ok(PepRequestMsg {
            inner: MsgRequestPrivateIdentity {
                creator: helper.creator,
                req_id: helper.req_id,
            },
        })
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct PepResponseWrapper {
    pub identity: String,
}
