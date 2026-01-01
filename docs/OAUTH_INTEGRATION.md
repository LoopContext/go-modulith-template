# OAuth Integration Guide

This document describes the OAuth/Social Login integration using the [markbates/goth](https://github.com/markbates/goth) library.

## Overview

The OAuth integration allows users to authenticate using external providers such as:
- **Google**
- **Facebook**
- **GitHub**
- **Apple**
- **Microsoft**
- **Twitter/X**

## Features

- **Login via OAuth**: Users can log in using their external provider accounts
- **Auto-link by email**: Automatically link external accounts to existing users with matching email addresses
- **Manual linking**: Users can link/unlink external accounts from their profile
- **Token encryption**: OAuth access and refresh tokens are encrypted before storage

## Architecture

```
┌─────────────┐     ┌────────────────┐     ┌──────────────────┐
│   Client    │────▶│  OAuth Handler │────▶│ External Provider│
│  (Browser)  │     │  (HTTP routes) │     │ (Google, GitHub) │
└─────────────┘     └────────────────┘     └──────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │ Auth Service │
                    │  (gRPC)      │
                    └──────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │  Repository  │
                    │  (Database)  │
                    └──────────────┘
```

## Configuration

### Environment Variables

```bash
# Enable OAuth
OAUTH_ENABLED=true
OAUTH_BASE_URL=https://api.example.com
OAUTH_AUTO_LINK_BY_EMAIL=true
OAUTH_TOKEN_ENCRYPTION_KEY=your-32-byte-encryption-key-here

# Google
OAUTH_GOOGLE_ENABLED=true
OAUTH_GOOGLE_CLIENT_ID=your-google-client-id
OAUTH_GOOGLE_CLIENT_SECRET=your-google-client-secret

# Facebook
OAUTH_FACEBOOK_ENABLED=true
OAUTH_FACEBOOK_CLIENT_ID=your-facebook-app-id
OAUTH_FACEBOOK_CLIENT_SECRET=your-facebook-app-secret

# GitHub
OAUTH_GITHUB_ENABLED=true
OAUTH_GITHUB_CLIENT_ID=your-github-client-id
OAUTH_GITHUB_CLIENT_SECRET=your-github-client-secret

# Microsoft
OAUTH_MICROSOFT_ENABLED=true
OAUTH_MICROSOFT_CLIENT_ID=your-microsoft-client-id
OAUTH_MICROSOFT_CLIENT_SECRET=your-microsoft-client-secret

# Twitter/X
OAUTH_TWITTER_ENABLED=true
OAUTH_TWITTER_CLIENT_ID=your-twitter-api-key
OAUTH_TWITTER_CLIENT_SECRET=your-twitter-api-secret

# Apple (requires additional setup)
OAUTH_APPLE_ENABLED=true
OAUTH_APPLE_CLIENT_ID=your-services-id
OAUTH_APPLE_TEAM_ID=your-team-id
OAUTH_APPLE_KEY_ID=your-key-id
OAUTH_APPLE_PRIVATE_KEY_PATH=/path/to/AuthKey.p8
```

### YAML Configuration

```yaml
auth:
  jwt_secret: "your-jwt-secret"
  oauth:
    enabled: true
    auto_link_by_email: true
    base_url: "https://api.example.com"
    token_encryption_key: "your-32-byte-encryption-key-here"
    providers:
      google:
        enabled: true
        client_id: "your-client-id"
        client_secret: "your-client-secret"
        scopes:
          - email
          - profile
      github:
        enabled: true
        client_id: "your-client-id"
        client_secret: "your-client-secret"
        scopes:
          - user:email
          - read:user
```

## Database Schema

The OAuth integration adds two new tables:

### `user_external_accounts`

Stores linked external provider accounts:

```sql
CREATE TABLE user_external_accounts (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,           -- google, facebook, github, etc.
    provider_user_id VARCHAR(255) NOT NULL,  -- ID from the provider
    email VARCHAR(255),
    name VARCHAR(255),
    avatar_url VARCHAR(512),
    access_token TEXT,                       -- Encrypted
    refresh_token TEXT,                      -- Encrypted
    token_expires_at TIMESTAMP,
    raw_data JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, provider_user_id)
);
```

### `oauth_states`

Temporary storage for OAuth state tokens:

```sql
CREATE TABLE oauth_states (
    state VARCHAR(255) PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    redirect_url TEXT,
    user_id VARCHAR(64),       -- Set when linking to existing user
    action VARCHAR(50) NOT NULL, -- login, link, signup
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);
```

## API Endpoints

### gRPC Service Methods

```protobuf
service AuthService {
  // Get list of enabled OAuth providers
  rpc GetOAuthProviders(GetOAuthProvidersRequest) returns (GetOAuthProvidersResponse);

  // Initiate OAuth flow (returns redirect URL)
  rpc InitiateOAuth(InitiateOAuthRequest) returns (InitiateOAuthResponse);

  // Complete OAuth flow (exchange code for tokens)
  rpc CompleteOAuth(CompleteOAuthRequest) returns (CompleteOAuthResponse);

  // Link external account to current user
  rpc LinkExternalAccount(LinkExternalAccountRequest) returns (LinkExternalAccountResponse);

  // Unlink external account from current user
  rpc UnlinkExternalAccount(UnlinkExternalAccountRequest) returns (UnlinkExternalAccountResponse);

  // List linked accounts for current user
  rpc ListLinkedAccounts(ListLinkedAccountsRequest) returns (ListLinkedAccountsResponse);
}
```

### HTTP Routes (via gRPC-Gateway)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/auth/oauth/providers` | List enabled OAuth providers |
| POST | `/v1/auth/oauth/initiate` | Start OAuth flow |
| GET | `/v1/auth/oauth/callback` | OAuth callback URL |
| POST | `/v1/auth/oauth/link` | Link external account |
| DELETE | `/v1/auth/oauth/link/{provider}` | Unlink external account |
| GET | `/v1/auth/oauth/accounts` | List linked accounts |

## OAuth Flow

### 1. Login Flow

```
Client                      Server                     Provider
  │                           │                           │
  │ GET /oauth/providers      │                           │
  │◀─────────────────────────▶│                           │
  │                           │                           │
  │ POST /oauth/initiate      │                           │
  │   {provider: "google"}    │                           │
  │◀─────────────────────────▶│                           │
  │                           │                           │
  │ Redirect to provider ─────┼───────────────────────────▶│
  │                           │                           │
  │ ◀──────────────────────────────────────────────────────│
  │     Callback with code    │                           │
  │                           │                           │
  │ GET /oauth/callback?code= │                           │
  │◀─────────────────────────▶│                           │
  │                           │                           │
  │ Receive JWT tokens        │                           │
  │◀──────────────────────────│                           │
```

### 2. Account Linking Flow

For authenticated users who want to link an external account:

```
Client                      Server                     Provider
  │                           │                           │
  │ POST /oauth/link          │                           │
  │   {provider: "github"}    │                           │
  │   [Authorization: Bearer] │                           │
  │◀─────────────────────────▶│                           │
  │                           │                           │
  │ Redirect to provider ─────┼───────────────────────────▶│
  │                           │                           │
  │ ◀──────────────────────────────────────────────────────│
  │     Callback with code    │                           │
  │                           │                           │
  │ GET /oauth/callback?code= │                           │
  │   state includes user_id  │                           │
  │◀─────────────────────────▶│                           │
  │                           │                           │
  │ Account linked success    │                           │
  │◀──────────────────────────│                           │
```

## Security Considerations

### Token Encryption

OAuth tokens (access_token and refresh_token) are encrypted using AES-256-GCM before storage:

```go
// Create encryptor with 32-byte key
enc, err := oauth.NewTokenEncryptor([]byte("your-32-byte-key"))

// Encrypt before storing
encryptedToken, err := enc.Encrypt(accessToken)

// Decrypt when needed
decryptedToken, err := enc.Decrypt(encryptedToken)
```

### State Token Validation

State tokens are:
- Cryptographically random
- Signed with HMAC-SHA256
- Short-lived (10 minutes by default)
- Single-use

### HTTPS Required

All OAuth redirects must use HTTPS in production. Configure your `base_url` accordingly.

## Provider-Specific Setup

### Google

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create or select a project
3. Enable the Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URI: `https://your-domain.com/v1/auth/oauth/callback?provider=google`

### GitHub

1. Go to [GitHub Developer Settings](https://github.com/settings/developers)
2. Create a new OAuth App
3. Set callback URL: `https://your-domain.com/v1/auth/oauth/callback?provider=github`

### Facebook

1. Go to [Facebook Developers](https://developers.facebook.com/)
2. Create a new app
3. Add Facebook Login product
4. Set callback URL: `https://your-domain.com/v1/auth/oauth/callback?provider=facebook`

### Apple

Apple Sign In requires additional setup:
1. Create an App ID with Sign In with Apple capability
2. Create a Services ID
3. Register your domain and callback URL
4. Create a private key for Sign In with Apple
5. Configure the key path in your settings

### Microsoft

1. Go to [Azure Portal](https://portal.azure.com/)
2. Navigate to Azure Active Directory > App registrations
3. Create a new registration
4. Add redirect URI: `https://your-domain.com/v1/auth/oauth/callback?provider=microsoftonline`

### Twitter/X

1. Go to [Twitter Developer Portal](https://developer.twitter.com/)
2. Create a new app
3. Enable OAuth 1.0a or 2.0
4. Set callback URL: `https://your-domain.com/v1/auth/oauth/callback?provider=twitter`

## Testing

Run OAuth-specific tests:

```bash
go test ./internal/oauth/... -v
```

## Troubleshooting

### Common Issues

1. **"Invalid redirect URI"**: Ensure your callback URL matches exactly what's configured in the provider console
2. **"State mismatch"**: Check that your JWT secret is consistent across instances
3. **"Token decryption failed"**: Verify the encryption key is exactly 32 bytes

### Debugging

Enable debug logging to see OAuth flow details:

```yaml
env: dev  # Enables detailed logging
```

Check the `oauth_states` table for pending authentication attempts.

## Migration Guide

If you're adding OAuth to an existing installation:

1. Run migrations:
   ```bash
   make migrate-up
   ```

2. Configure environment variables

3. Restart the server

4. Verify providers are enabled:
   ```bash
   curl https://api.example.com/v1/auth/oauth/providers
   ```

