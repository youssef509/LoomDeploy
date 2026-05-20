# Security Policy

## Supported Versions

| Version | Supported |
|---|---|
| latest (`main`) | ✅ |
| older releases | ❌ |

LoomDeploy is currently in **alpha**. Only the latest `main` branch receives security updates.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

If you discover a security vulnerability, please report it responsibly:

1. Go to the [Security tab](https://github.com/youssef509/loomdeploy/security/advisories/new) of this repository
2. Click **"Report a vulnerability"** to open a private advisory
3. Describe the vulnerability in detail, including:
   - Steps to reproduce
   - Potential impact
   - Any suggested fix (optional)

You will receive a response within **72 hours**. We will work with you to understand the issue, develop a fix, and coordinate disclosure.

## Security Considerations

- **JWT Secret** — The `JWT_SECRET` in your `.env` file protects all authenticated API routes. Use a strong randomly-generated value and never commit it to version control.
- **Docker socket** — LoomDeploy proxies the Docker socket through nginx for isolation. Never expose the raw Docker socket.
- **Webhook URLs** — Webhook secrets are included in the URL. Treat them like passwords.
- **First-run registration** — Registration is open only for the first user. After the admin account is created, new users must be invited by an admin.
- **Database** — LoomDeploy uses a local SQLite file. Ensure `/var/lib/loomdeploy/` is not publicly accessible.
