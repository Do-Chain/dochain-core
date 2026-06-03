# DoChain MFA

This module enforces MFA on-chain for protected transactions. The authenticator
app code is verified off-chain by the DoChain MFA service. The chain only
verifies a short-lived approval signature from the registered MFA approval key.

## Protected Actions

The antehandler requires MFA approval when an enabled account performs any of
these actions with `udo`:

- `cosmos.bank.v1beta1.MsgSend`
- `cosmos.bank.v1beta1.MsgMultiSend`
- `cosmos.staking.v1beta1.MsgUndelegate`
- `cosmos.staking.v1beta1.MsgBeginRedelegate`
- `ibc.applications.transfer.v1.MsgTransfer`
- wasm execute messages with `udo` funds
- DoChain market swap/swap-send messages spending `udo`
- the same protected messages inside `cosmos.authz.v1beta1.MsgExec`

## Wallet Flow

1. Wallet asks the MFA service to create a TOTP secret and MFA approval key.
2. User scans the TOTP QR code in an authenticator app.
3. Wallet submits an MFA enable control memo with the new approval public key,
   a valid approval from that new key, and the wallet account signature. This
   can be attached to a normal setup transaction until dedicated
   `MsgEnableMFA`/`MsgDisableMFA` protobuf messages are added.
4. For protected transactions, the wallet sends the unsigned transaction details
   and the user's authenticator code to the MFA service.
5. The MFA service validates the code and signs the canonical approval payload.
6. Wallet puts the approval in the transaction memo and broadcasts the signed tx.
7. Validators reject the tx unless the approval is valid for that exact tx.

## Memo Format

The MFA data lives under the `dochain_mfa` memo key:

```json
{
  "dochain_mfa": {
    "approvals": [
      {
        "account": "do1...",
        "expires_at": 1700000300,
        "signature": "base64-signature"
      }
    ]
  }
}
```

Enable and disable control actions use the same memo key:

```json
{
  "dochain_mfa": {
    "approvals": [
      {
        "account": "do1...",
        "expires_at": 1700000300,
        "signature": "base64-signature"
      }
    ],
    "enable": {
      "account": "do1...",
      "approval_pub_key": "base64-secp256k1-pubkey"
    }
  }
}
```

```json
{
  "dochain_mfa": {
    "approvals": [
      {
        "account": "do1...",
        "expires_at": 1700000300,
        "signature": "base64-signature"
      }
    ],
    "disable": {
      "account": "do1..."
    }
  }
}
```

Initial enable requires both the wallet account signature and a valid approval
signature from the new MFA key after the user completes the authenticator flow.
Disabling MFA or rotating the approval key requires a valid existing MFA
approval.

## Approval Payload

The MFA service signs this canonical JSON object with the registered secp256k1
approval key:

```json
{
  "version": "dochain-mfa-v1",
  "chain_id": "dochain-1",
  "account": "do1...",
  "expires_at": 1700000300,
  "timeout_height": 0,
  "messages_hash": "hex-sha256",
  "signers": [
    {
      "address": "do1...",
      "sequence": 7
    }
  ]
}
```

`messages_hash` is the SHA-256 hash of the sorted JSON encoding of every tx
message. The full approval payload is sorted before signing. The signer list and
sequences are derived from current on-chain account state, so the approval cannot
be replayed after the tx signer sequence changes.
