use std::collections::HashMap;
use cosmwasm_std::{
    entry_point, to_json_binary, Binary, CosmosMsg, Deps, DepsMut, Env, MessageInfo, Reply,
    Response, StdResult, SubMsg, Order,
};
use prost::Message;
use crate::error::ContractError;
use crate::msg::{
    AllIdentitiesResponse, ExecuteMsg, IdentityResponse, InstantiateMsg, PrivateDecryptionKey,
    PepRequestMsg, QueryMsg,
};
use crate::state::{
    IdentityRecord, PendingRequest, LAST_REPLY_ID, PENDING_REQUESTS, PUBKEY, RECORDS, REQUESTER,
};
use fairblock_proto::fairyring::pep::{
    MsgRequestPrivateDecryptionKey, MsgRequestPrivateDecryptionKeyResponse,
    MsgRequestPrivateIdentity, MsgRequestPrivateIdentityResponse,
};

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response<PepRequestMsg>, ContractError> {
    LAST_REPLY_ID.save(deps.storage, &0)?;
    PUBKEY.save(deps.storage, &msg.pubkey)?;
    Ok(Response::new())
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response<PepRequestMsg>, ContractError> {
    match msg {
        ExecuteMsg::UpdatePubkey { pubkey } => update_pubkey(deps, env, info, pubkey),
        ExecuteMsg::RequestPrivateKeyshare { identity, secp_pubkey } => {
            execute_request_keyshare(deps, env, info, identity, secp_pubkey)
        }
        ExecuteMsg::RequestIdentity { authorized_address } => {
            execute_request_identity(deps, env, info, authorized_address)
        }
        ExecuteMsg::StoreEncryptedData { identity, data } => {
            store_encrypted_data(deps, env, info, identity, data)
        }
        ExecuteMsg::ExecuteContractPrivateMsg { identity, private_decryption_key } => {
            execute_private_keys(deps, env, info, identity, private_decryption_key)
        }
    }
}

fn update_pubkey(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    pubkey: String,
) -> Result<Response<PepRequestMsg>, ContractError> {
    PUBKEY.save(deps.storage, &pubkey)?;
    Ok(Response::new())
}

fn execute_private_keys(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    identity: String,
    dec_keys: PrivateDecryptionKey,
) -> Result<Response<PepRequestMsg>, ContractError> {
    let mut record = RECORDS.load(deps.storage, identity.as_str())?;
    let requester = REQUESTER.load(deps.storage)?;
    record.private_keyshares.insert(requester.clone(), dec_keys.private_keyshares);
    RECORDS.save(deps.storage, identity.as_str(), &record)?;
    Ok(Response::new()
        .add_attribute("action", "store_encrypted_keyshares")
        .add_attribute("identity", identity)
        .add_attribute("requester_address", dec_keys.requester))
}

fn execute_request_keyshare(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    identity: String,
    secp_pubkey: String,
) -> Result<Response<PepRequestMsg>, ContractError> {
    let mut record = RECORDS.load(deps.storage, identity.as_str())?;
    if info.sender.to_string() != record.creator {
        return Err(ContractError::Unauthorized {});
    }
    let now = env.block.time.seconds();
    let thirty_days: u64 = 60 * 60 * 24 * 30;
    if now <= record.last_submission + thirty_days {
        return Err(ContractError::NotYetDue {});
    }
    let contract_addr = env.contract.address.to_string();
    let mut reply_id = LAST_REPLY_ID.load(deps.storage)?;
    reply_id += 1;
    LAST_REPLY_ID.save(deps.storage, &reply_id)?;
    let pending = PendingRequest {
        creator: info.sender.to_string(),
        authorized_address: info.sender.to_string(),
    };
    PENDING_REQUESTS.save(deps.storage, reply_id, &pending)?;
    record.private_keyshares.insert(info.sender.to_string(), Vec::new());
    RECORDS.save(deps.storage, identity.as_str(), &record)?;
    REQUESTER.save(deps.storage, &info.sender.to_string())?;
    let msg = MsgRequestPrivateDecryptionKey {
        creator: contract_addr,
        identity,
        secp_pubkey,
    };
    let e = msg.encode_to_vec();
    let d = Binary::new(e);
    let any_msg = cosmwasm_std::AnyMsg {
        type_url: "/fairyring.pep.MsgRequestPrivateDecryptionKey".to_string(),
        value: d,
    };
    let cosmos_msg = CosmosMsg::Any(any_msg);
    let sub_msg = SubMsg::reply_on_success(cosmos_msg, reply_id);
    Ok(Response::new()
        .add_submessage(sub_msg)
        .add_attribute("action", "request_private_keyshare")
        .add_attribute("pending_reply_id", reply_id.to_string()))
}

fn execute_request_identity(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    authorized_address: String,
) -> Result<Response<PepRequestMsg>, ContractError> {
    let mut reply_id = LAST_REPLY_ID.load(deps.storage)?;
    reply_id += 1;
    LAST_REPLY_ID.save(deps.storage, &reply_id)?;
    let contract_addr = env.contract.address.to_string();
    let pending = PendingRequest {
        creator: info.sender.to_string(),
        authorized_address: authorized_address.clone(),
    };
    PENDING_REQUESTS.save(deps.storage, reply_id, &pending)?;
    let req_id = format!("req-{}-{}", info.sender.to_string(), reply_id);
    let inner_msg = MsgRequestPrivateIdentity {
        creator: contract_addr.clone(),
        req_id,
    };
    let e = inner_msg.encode_to_vec();
    let d = Binary::new(e);
    let any_msg = cosmwasm_std::AnyMsg {
        type_url: "/fairyring.pep.MsgRequestPrivateIdentity".to_string(),
        value: d,
    };
    let cosmos_msg = CosmosMsg::Any(any_msg);
    let sub_msg = SubMsg::reply_on_success(cosmos_msg, reply_id);
    Ok(Response::new()
        .add_submessage(sub_msg)
        .add_attribute("action", "request_identity")
        .add_attribute("pending_reply_id", reply_id.to_string()))
}

fn store_encrypted_data(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    identity: String,
    data: String,
) -> Result<Response<PepRequestMsg>, ContractError> {
    let mut record = RECORDS.load(deps.storage, identity.as_str())?;
    record.encrypted_data = data.clone();
    record.last_submission = env.block.time.seconds();
    RECORDS.save(deps.storage, identity.as_str(), &record)?;
    Ok(Response::new()
        .add_attribute("action", "store_data")
        .add_attribute("identity", identity)
        .add_attribute("data", data))
}

#[entry_point]
pub fn reply(
    deps: DepsMut,
    _env: Env,
    msg: Reply,
) -> Result<Response<PepRequestMsg>, ContractError> {
    let reply_id = msg.id;
    let pending = PENDING_REQUESTS
        .load(deps.storage, reply_id)
        .map_err(|_| ContractError::PendingRequestNotFound { id: reply_id })?;
    let submsg_result = msg
        .result
        .into_result()
        .map_err(|e| ContractError::ReplyError { error: e.to_string() })?;
    let binary_data = submsg_result
        .msg_responses
        .get(0)
        .map(|resp| resp.value.clone())
        .ok_or(ContractError::ReplyMissingData {})?;
    let identity_result = MsgRequestPrivateIdentityResponse::decode(binary_data.as_slice());
    let keyshare_result = MsgRequestPrivateDecryptionKeyResponse::decode(binary_data.as_slice());
    if let Ok(pep_response) = identity_result {
        if !pep_response.identity.is_empty() {
            let pubkey = PUBKEY.load(deps.storage)?;
            let identity = pep_response.identity;
            let record = IdentityRecord {
                identity: identity.clone(),
                pubkey,
                creator: pending.authorized_address,
                encrypted_data: "".to_string(),
                private_keyshares: HashMap::new(),
                last_submission: 0,
            };
            RECORDS.save(deps.storage, identity.as_str(), &record)?;
            PENDING_REQUESTS.remove(deps.storage, reply_id);
            return Ok(Response::new()
                .add_attribute("action", "store_identity")
                .add_attribute("identity", identity));
        }
    }
    if let Ok(_keyshare_response) = keyshare_result {
        return Ok(Response::new().add_attribute("action", "request_private_keyshare"));
    }
    Err(ContractError::ReplyError {
        error: "Unknown response type".to_string(),
    })
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::GetIdentity { identity } => to_json_binary(&query_identity(deps, identity)?),
        QueryMsg::GetAllIdentity {} => to_json_binary(&query_all_identities(deps)?),
    }
}

fn query_identity(deps: Deps, identity: String) -> StdResult<IdentityResponse> {
    let record = RECORDS.load(deps.storage, identity.as_str())?;
    Ok(IdentityResponse { record })
}

fn query_all_identities(deps: Deps) -> StdResult<AllIdentitiesResponse> {
    let records: Vec<IdentityRecord> = RECORDS
        .range(deps.storage, None, None, Order::Ascending)
        .map(|item| item.map(|(_, record)| record))
        .collect::<StdResult<Vec<_>>>()?;
    Ok(AllIdentitiesResponse { records })
}
