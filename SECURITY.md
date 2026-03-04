# Security Policy

## Supported Versions
This repository currently supports security fixes on the `main` branch.

## Reporting a Vulnerability
Please do not open public GitHub issues for suspected security vulnerabilities.

Report privately by contacting the maintainers through GitHub security advisories:
1. Open the repository's `Security` tab.
2. Create a private vulnerability report.
3. Include reproduction steps, impact, and affected commit/branch when possible.

## Response Expectations
- Initial triage target: within 72 hours.
- Mitigation plan target: within 7 days for confirmed high/critical findings.
- Fixes are released as soon as validation is complete.

## Security Controls in CI
- `govulncheck` for Go dependency and code-path vulnerability checks
- `npm audit` for client dependencies
- `gitleaks` for secrets detection
