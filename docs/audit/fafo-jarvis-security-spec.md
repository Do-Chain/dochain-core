# FAFO And Jarvis Security Specification

Status: draft for implementation planning

Scope:
- `dochain-core` consensus/runtime security controls
- wallet and MFA approval service controls that protect account actions
- oracle and validator operations signals that should feed alerting

This document makes the FAFO and Jarvis labels concrete. Anything that changes
consensus behavior, state layout, ante handling, module params, or transaction
validity must ship through a normal chain software upgrade. Off-chain service
controls can ship independently with rolling service restarts.

## FAFO Security Measures

FAFO is the account and operator protection layer. Its job is to make high-risk
actions require the right proof, from the right actor, at the right time, and to
fail closed when proof is missing or malformed.

### Protected Actions

The first implementation must cover:

- bank sends from MFA-enabled accounts
- IBC transfers from MFA-enabled accounts
- delegate, redelegate, undelegate, and cancel-unbonding actions
- MFA enable, disable, approval-key rotation, guardian setup, recovery start,
  recovery cancel, and recovery execute actions
- validator operator changes that alter validator security posture, including
  commission changes, consensus key rotation, and validator metadata updates
- oracle feeder key import/export and vote submission service startup

### Required Proofs

MFA-enabled account spends and staking actions require:

- normal transaction signatures from the account signers
- a short-lived MFA approval signature bound to chain ID, account, timeout
  height, message hash, signer sequence data, and action type

MFA control actions require:

- existing MFA approval for disable, rotate, set guardian, start recovery, and
  cancel recovery
- guardian approval only for disable or rotate
- delayed recovery execution only after the configured delay and only for the
  pending recovery action
- recovery-code approvals only for recovery-control transactions, never for
  arbitrary spend, IBC, delegate, redelegate, or undelegate messages

Validator and oracle operator actions require:

- operator key signature
- explicit operator confirmation in CLI or service config
- audit log entry with actor, action, chain ID, height or wall-clock time, and
  config hash

### Fail-Closed Rules

Implementations must reject the transaction or service startup when:

- MFA policy bytes exist but cannot be decoded
- an approval public key is malformed or has the wrong length
- approval signatures are expired, malformed, or bound to a different message
  hash, signer sequence, account, chain ID, action, or timeout height
- a transaction attempts more than one MFA control action
- a recovery transaction contains any non-recovery message
- the MFA approval service cannot reach its encrypted secret store or signer
- the oracle feeder cannot load a v2 encrypted keystore in production mode

### State And Secret Handling

On-chain state:

- store MFA policy state using protobuf encoding, not JSON
- surface decode errors instead of treating corrupted state as no policy
- include module migration tests for any state encoding change

Off-chain MFA service:

- generate TOTP secrets server-side
- store TOTP secrets and approval signing keys encrypted at rest
- keep approval signing keys in KMS, HSM, or a separate signer process when
  available
- never overwrite an active MFA service record without current MFA approval,
  guardian recovery, or proof that no active on-chain policy exists
- record append-only audit events for setup, approval, disable, rotate, guardian,
  and recovery actions

Oracle feeder:

- refuse legacy weak keystores in production mode after a migration window
- require at least one private or TLS-protected data source URL in production
- support a median or quorum mode before using multiple price-server URLs as a
  security boundary

### Acceptance Tests

Before FAFO is treated as complete:

- invalid wallet signatures cannot enable, disable, rotate, start recovery,
  cancel recovery, or execute recovery
- recovery-code approval cannot approve bank send, IBC transfer, delegate,
  redelegate, undelegate, or cancel-unbonding transactions
- malformed MFA policy bytes fail closed
- guardian approval cannot approve normal protected spends
- delayed recovery cannot execute before its height
- one transaction cannot include multiple MFA control actions
- oracle feeder production mode refuses legacy keystores
- the MFA service cannot create an active setup from a client-supplied TOTP
  secret

## Jarvis Anomaly Detector

Jarvis is the monitoring and alerting layer. It must not decide consensus. It
observes chain, wallet, oracle, and validator signals, assigns severity, and
routes actionable alerts.

### Signal Sources

Jarvis should ingest:

- block height, block time, missed blocks, validator voting power, and validator
  set changes
- governance proposal creation, deposit, vote, tally, cancel, and upgrade-plan
  events
- MFA enable, disable, rotate, guardian, recovery start, recovery cancel, and
  recovery execute events
- bank send, IBC transfer, delegate, redelegate, undelegate, and
  cancel-unbonding events involving MFA-enabled accounts
- oracle exchange-rate votes, feeder liveness, feeder source count, price-source
  freshness, and max source deviation
- wallet MFA approval attempts, failures, lockouts, recovery-code use, and setup
  events
- node RPC/LCD health, peer count, mempool size, and disk usage

### Detection Rules

Critical alerts:

- scheduled chain upgrade height is approaching and a validator is still running
  the old binary
- halt, consensus failure, or more than one missed block window on a validator
- MFA recovery execution for a high-value or validator-linked account
- recovery-code approval requested for a non-recovery transaction
- oracle feeder publishes a price with source deviation above configured limit
- validator consensus key changes unexpectedly

High alerts:

- MFA disable, approval-key rotation, guardian change, or recovery start
- burst of failed MFA approvals by account or IP
- oracle feeder source count drops below quorum
- validator commission or metadata changes outside maintenance window
- governance proposal includes software upgrade, parameter change, community
  pool spend, or module authority change

Medium alerts:

- price-source freshness nearing max age
- peer count below target
- RPC/LCD error rate above threshold
- deposit surge on a governance proposal
- repeated transaction failures from the same wallet session

### Alert Payload

Every Jarvis alert must include:

- `id`: stable event ID
- `severity`: `critical`, `high`, `medium`, or `low`
- `source`: `chain`, `wallet`, `oracle`, `validator`, or `governance`
- `chain_id`
- `height` when available
- `account`, `validator`, `proposal_id`, or `denom` when relevant
- `rule`
- `summary`
- `evidence`: bounded structured context, never raw secrets
- `recommended_action`
- `created_at`

### Outputs

Minimum outputs:

- structured JSON logs
- Prometheus counters and gauges
- webhook delivery for critical and high alerts

Recommended outputs:

- Slack or Discord webhook
- email or pager integration for validator operators
- wallet notification feed for MFA recovery and guardian events

### Rollout

Phase 1:

- implement off-chain Jarvis collector and alert rules only
- no consensus changes
- no automated transaction submission

Phase 2:

- add indexer-backed account risk views for wallet UI
- add operator dashboard for validators and oracle feeders

Phase 3:

- if governance approves, add on-chain params/events that improve observability
- ship those changes only through a planned software upgrade

## Live Chain Upgrade Boundary

No FAFO or Jarvis patch may be deployed directly to live validators when it
changes consensus behavior. Consensus-affecting changes require:

- focused unit tests
- full package tests in a Go-enabled environment
- testnet or fork rehearsal
- versioned binary build
- governance software-upgrade proposal
- validator installation instructions before upgrade height
- rollback and halt-response plan

Off-chain service-only changes, such as oracle price-server TLS fixes,
dependency updates, MFA service secret storage hardening, and Jarvis collectors,
can deploy with rolling restarts after staging verification.
