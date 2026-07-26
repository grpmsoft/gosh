# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x (latest beta) | ✅ |
| < 0.1.0 | ❌ |

## Reporting a Vulnerability

**Please do NOT report security vulnerabilities through public GitHub issues.**

Instead, report them through GitHub's private vulnerability reporting:

1. Go to [github.com/grpmsoft/gosh/security/advisories](https://github.com/grpmsoft/gosh/security/advisories)
2. Click **"Report a vulnerability"**
3. Fill in the details

Alternatively, email **security@grpmsoft.com** with:

- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Impact assessment (if known)

### Response Timeline

- **Acknowledgment**: within 48 hours
- **Initial assessment**: within 7 days
- **Fix or mitigation**: depends on severity, typically within 30 days

## Security Considerations

GoSh is a shell that executes commands with the privileges of the running user. Key areas:

- **Command execution**: GoSh uses `os/exec` for external commands and `mvdan.cc/sh` for POSIX script interpretation
- **Environment variables**: accessible and modifiable through built-in commands (`export`, `env`)
- **File system**: accessed through standard Go `os` package operations
- **No network services**: GoSh does not listen on any network ports
- **No elevated privileges**: GoSh never requests or uses elevated privileges beyond the current user

## Scope

The following are **in scope** for security reports:

- Command injection through parser/lexer bypasses
- Path traversal in built-in commands (cd, file completion)
- Environment variable leakage
- History file permission issues
- Unexpected code execution through script detection

The following are **out of scope**:

- Vulnerabilities in commands executed by the user (GoSh passes through to the OS)
- Issues requiring physical access to the machine
- Social engineering attacks
- Denial of service through resource exhaustion (shell is a local tool)
