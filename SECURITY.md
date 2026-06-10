# Security Policy

## Reporting A Vulnerability

Please do not open public issues for suspected vulnerabilities in chain,
validator, wallet, oracle, governance, or MFA code.

Email the maintainers with:

- affected repository, component, version, commit SHA, or binary hash
- chain ID and network, if the issue affects a live deployment
- steps to reproduce or proof of concept
- expected impact
- logs, traces, screenshots, or transaction hashes that help triage

The maintainers should acknowledge reports within 3 business days and coordinate
fixes privately before public disclosure.

## Supported Versions

Security fixes are prioritized for:

- the active `main` branch
- any release branch or binary currently recommended to DoChain validators
- any live-chain upgrade branch between proposal approval and upgrade execution

Consensus-affecting fixes must be released through the chain's normal software
upgrade process unless the validator set coordinates an emergency response.
