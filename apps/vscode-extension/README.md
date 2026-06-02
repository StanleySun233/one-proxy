# OneProxy VS Code Extension

This is the minimum Remote-SSH integration skeleton.

Commands:

- `OneProxy: Login`
- `OneProxy: Write SSH Config`
- `OneProxy: Connect with Remote-SSH`

The extension stores panel session tokens in VS Code SecretStorage. The generated SSH config uses the `oneproxy proxy-command` helper and writes the proxy token to `~/.config/oneproxy/proxy-token` with `0600` permissions instead of embedding it in `~/.ssh/config`.

Current scope:

- Login through `/api/v1/auth/login`
- Generate a single SSH host block
- Call VS Code Remote-SSH through `vscode.openFolder`

Future scope:

- Fetch panel access paths instead of prompting host and port manually
- Refresh expired proxy tokens
- Manage multiple host aliases
