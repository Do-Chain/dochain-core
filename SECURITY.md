# Security Policy

## Supported Branches

Security fixes are prioritized for the active release branch and the current mainline development branch.

## Reporting a Vulnerability

Please do not open public issues for suspected vulnerabilities. Report privately to the Do-Chain maintainers with:

- affected repository and commit
- reproduction steps or proof of impact
- affected module, endpoint, or deployment path
- any suggested mitigation

Rotate any exposed credentials immediately. Do not include private keys, mnemonics, validator keys, oracle feeder keys, or server passwords in issues, pull requests, logs, or chat transcripts.

## Security Defaults

Production deployments should build from an audited GitHub commit or signed release artifact. RPC, LCD, gRPC, metrics, and admin endpoints must bind to localhost or a private network unless deliberately fronted by a rate-limited proxy. P2P may be public when required for node operation.

Validator and oracle credentials must be supplied from a secret manager or mounted secret file. Public localnet test mnemonics must never be reused for funded accounts, validators, oracle feeders, or production test nodes.
