# Security policy

## Report a vulnerability

Do not open a public issue for a suspected vulnerability.

Use GitHub's private vulnerability-reporting form:

https://github.com/blackstork-io/blackstork-cli/security/advisories/new

Include enough information for us to reproduce and assess the issue:

- affected version or commit;
- affected operating system and architecture;
- required configuration or template;
- reproduction steps or a minimal proof of concept;
- observed and expected behavior;
- potential impact; and
- any suggested mitigation.

Remove production data, access tokens, private URLs, and customer information
from the report. If a secret was exposed while testing, revoke it before
submitting the report.

We will acknowledge the report, investigate it, and coordinate disclosure and
remediation with the reporter. Please do not disclose the issue publicly until
users have had a reasonable opportunity to update.

## Supported versions

Security fixes are normally released for the latest published version. We may
ask reporters to reproduce an issue against the current release or `main` when
the affected code has changed.

## Scope

Relevant issues include vulnerabilities in the CLI, parser, evaluation engine,
plugin protocol, official plugins, release artifacts, and repository-owned
automation.

For ordinary bugs and feature requests, use the public issue tracker.
