# Things Cloud MCP

An MCP server that connects AI assistants to Things 3 via Things Cloud.

**Public endpoint: https://thingscloudmcp.com**

Add this URL to your MCP client and start managing Things 3 tasks with AI. Multi-user — each user authenticates with their own Things Cloud credentials.

## Features

- Streamable HTTP transport with OAuth 2.1 and Basic authentication
- Multi-user support with per-user operation serialization and credential-bound sessions
- 23 tools for reading and managing tasks, projects, headings, areas, tags, and checklists
- Real-time sync with Things 3 apps on Mac, iPhone, and iPad
- MCP output schemas and structured content for every tool
- Fail-closed cursor/schema validation, uncertain-write reconciliation without automatic retry, and explicit confirmation for permanent deletion

## Self-hosting

If you prefer to host your own instance:

```bash
go build -o things-mcp .
./things-mcp
```

The server listens on port 8080 by default (set `PORT` to override). Optionally set `JWT_SECRET` for stable tokens across restarts. Things passwords stored for OAuth are encrypted with AES-GCM; set a durable high-entropy `CREDENTIALS_SECRET`, or persist the generated `DATA_DIR/credentials.key` alongside `oauth.db`. Refresh tokens are stored as hashes.

The server makes `DATA_DIR` private to itself (mode `0700`). When it can't, because it doesn't own the directory, as with a Kubernetes volume shared with the pod through `fsGroup`, it uses the directory as it is unless anyone can write to it. The files it keeps there are private to it either way.

- OAuth clients (Claude.ai, ChatGPT) authenticate via the built-in OAuth 2.1 flow
- CLI clients (Claude Code, Cursor, Windsurf) use Basic auth headers

### Single-user mode

Behind something else that signs people in, such as an MCP proxy with its own OAuth, set `THINGS_EMAIL` and `THINGS_PASSWORD` to one Things Cloud account. Every request then acts as that account, with or without an `Authorization` header. The server serves no OAuth endpoints and keeps nothing in `DATA_DIR`. Since whoever reaches it acts as the account, it listens only on `127.0.0.1` unless `HOST` names another address. `ALLOWED_EMAILS` and `READ_ONLY` still apply, so the server refuses to start with an account outside `ALLOWED_EMAILS`.

### Deploy to Fly.io

The included `Dockerfile` stores OAuth state and generated keys under `/data`. To deploy on [Fly.io](https://fly.io), create a `fly.toml` for the app and mount the volume at `/data`:

```bash
brew install flyctl
fly auth login

fly launch --no-deploy                   # generate and review fly.toml
fly volumes create data --size 1         # 1 GB persistent volume for tokens and JWT secret
fly secrets set JWT_SECRET=$(openssl rand -hex 32)
fly secrets set CREDENTIALS_SECRET=$(openssl rand -hex 32)
fly deploy
```

**Important:** The persistent volume is required. Without it, OAuth state and any generated credential-encryption key are stored on ephemeral disk. Losing the encryption key makes persisted credentials intentionally unreadable. Back up `oauth.db` and the matching key together before deployment or migration.

This project writes through a reverse-engineered, unofficial Things Cloud protocol. Keep a restorable Things for Mac backup and use a disposable account for write validation. Production health checks should remain read-only unless a real write is explicitly intended.

Your endpoint will normally be `https://<app-name>.fly.dev`; verify current Fly configuration and pricing before deployment.

Built with [things-cloud-sdk](https://github.com/arthursoares/things-cloud-sdk) and [mcp-go](https://github.com/mark3labs/mcp-go).
