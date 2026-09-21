# OIDC service identity

Пакет `pkg/oidc` отделён от пользовательской/session-авторизации.

- `ClientCredentialsProvider` получает service token и кеширует его до безопасного окна перед expiry; параллельные callers разделяют один refresh.
- `Verifier` получает и кеширует JWKS, поддерживает RSA и Ed25519 keys и проверяет issuer, audience, allowlist алгоритмов и caller (`azp`/`client_id`).
- `pkg/grpckit` содержит bearer interceptors для unary и streaming RPC.

Production service должен включать verifier/client явно. Development bypass является opt-in настройкой приложения и не должен автоматически зависеть только от hostname.
