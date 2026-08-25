# Security policy

## Supported versions

The latest commit on `main` receives security fixes.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Use GitHub's private vulnerability reporting feature under the repository Security tab. Include affected versions, reproduction steps, impact, and any suggested mitigation.

You should receive an acknowledgment within 5 business days. Please allow time for validation and a coordinated fix before public disclosure.

## Deployment guidance

- Place authentication, TLS, and caller rate limits at a trusted ingress.
- Store API keys in a secret manager and rotate them regularly.
- Restrict outbound traffic to configured provider hosts.
- Run the container as non-root with a read-only filesystem.
- Protect `/metrics` and readiness endpoints from public access.
- Treat prompts, completions, and tool arguments as sensitive data.

The repository's [threat model](docs/THREAT_MODEL.md) documents current assumptions and residual risks.
