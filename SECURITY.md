# Security Policy

## Supported Versions

Currently supported versions of go-create:

| Version | Supported          |
| ------- | ------------------ |
| main    | Yes                |
| < 1.0   | Development only   |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability in go-create, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report security issues by:

1. Opening a private security advisory on GitHub
2. Or emailing the maintainers directly

### What to Include

Please provide:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if you have one)
- Your contact information

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity
  - Critical: Within 7 days
  - High: Within 30 days
  - Medium: Within 90 days
  - Low: Best effort

## Security Considerations

### Password Handling

go-create implements several security measures for password handling:

1. **Password Masking**: Passwords are masked in logs and output
2. **Secure Temporary Files**: Uses `os.CreateTemp()` with mode 0600
3. **Automatic Cleanup**: Temporary files are cleaned up on exit
4. **Password Policy**: Enforces strong passwords for new users (30+ chars, mixed case, digits, special chars)
5. **Error Sanitization**: Passwords removed from error messages

### Connection Security

1. **No Plain Text Storage**: Passwords not stored in plain text
2. **Secure Config Files**: Config files use 0600 permissions
3. **Multiple Auth Methods**: Supports .my.cnf, config files, environment
4. **DSN Sanitization**: Connection strings sanitized in logs

### Known Limitations

1. **MySQL CLI Execution**: Uses `mysql` command-line client which may expose passwords in process listings briefly
2. **Admin Credentials**: Tool requires admin credentials to create users
3. **File-Based Auth**: Temporary files created for complex passwords (cleaned up automatically)

### Mitigations for Limitations

1. **Use SQL File Mode**: For complex passwords, use `-use-sql-file` flag
2. **Secure Environment**: Run in a controlled environment
3. **Credential Rotation**: Rotate admin credentials regularly
4. **Audit Logs**: Enable MySQL audit logging for production use

## Best Practices

### For Users

1. **Use Strong Admin Passwords**: Admin account should have a strong password
2. **Limit Network Access**: Restrict MySQL network access
3. **Use SSL/TLS**: Enable encrypted connections to MySQL
4. **Rotate Credentials**: Regularly rotate passwords
5. **Monitor Access**: Enable and review MySQL audit logs
6. **Use Config Files**: Store credentials in config files with 0600 permissions
7. **Avoid Command Line Passwords**: Use config files or .my.cnf instead of -p flag

### For Developers

1. **Never Log Passwords**: Use the sanitization functions
2. **Secure Temp Files**: Use `os.CreateTemp()` not predictable paths
3. **Check Errors**: Always handle errors from security-sensitive operations
4. **Sanitize Errors**: Remove passwords from error messages
5. **Use Defer for Cleanup**: Ensure temp files are cleaned up
6. **Validate Inputs**: Check all user inputs for safety

## Security Features

### Implemented (Phase 1)

- Password masking in all log output
- Secure temporary file handling with `os.CreateTemp()`
- Automatic cleanup of sensitive files
- Error message sanitization utilities
- No deprecated packages (removed io/ioutil)

### Implemented (Phase 2)

- Security-focused linting with gosec
- Comprehensive test coverage for auth and config packages
- Error handling improvements
- Code quality standards enforced

### Planned

- Audit logging of all operations
- Support for HashiCorp Vault integration
- Support for AWS Secrets Manager
- Enhanced connection encryption options
- Role-based access for the tool itself

## Threat Model

### In Scope

- Password exposure in logs
- Temporary file security
- SQL injection in user/role names
- Credential theft from config files
- Process memory attacks

### Out of Scope

- MySQL server vulnerabilities
- Network-level attacks
- Operating system vulnerabilities
- Physical access attacks
- Social engineering

## Security Updates

Security updates will be released as soon as possible after a vulnerability is confirmed. Updates will be announced via:

1. GitHub Security Advisories
2. Release notes
3. Project README

## Acknowledgments

We appreciate responsible disclosure of security vulnerabilities. Contributors who report valid security issues will be acknowledged (with their permission) in:

- Security advisories
- Release notes
- Project documentation

## Contact

For security concerns, please contact the maintainers through GitHub's private vulnerability reporting feature.

---

Last Updated: 2026-04-27
