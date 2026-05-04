# Lost Pet Backend (Monolith)

Backend em Go para cadastro de relatos de animais encontrados/perdidos, autenticação OAuth2 social e comunicação interna entre usuários.

## Arquitetura

- Monólito em camadas (`domain`, `usecase`, `interface/http`, `infra`)
- MongoDB como banco principal
- Autenticação via OAuth2 (Google) + JWT de sessão da API
- Mensageria interna assíncrona vinculada a relatos

## Variáveis de ambiente

- `PORT` (default: `8080`)
- `MONGO_URI` (default: `mongodb://localhost:27017`)
- `MONGO_DATABASE` (default: `petfinder`)
- `JWT_SECRET` (default: `dev-secret`)
- `GOOGLE_CLIENT_ID`
- `GOOGLE_CLIENT_SECRET`
- `GOOGLE_REDIRECT_URL`

## Executar

```bash
go run ./cmd/api
```

## Endpoints principais

- `GET /health`
- `GET /swagger/` (Swagger UI)
- `GET /v1/auth/oauth/google/start`
- `GET /v1/auth/oauth/google/callback`
- `POST /v1/auth/refresh`
- `POST /v1/auth/logout`
- `GET /v1/users/me`
- `PATCH /v1/users/me`
- `DELETE /v1/users/me`
- `POST /v1/reports`
- `GET /v1/reports/{id}`
- `GET /v1/reports`
- `PATCH /v1/reports/{id}`
- `POST /v1/reports/{id}/resolve`
- `POST /v1/reports/{id}/archive`
- `POST /v1/reports/{id}/conversations`
- `GET /v1/conversations`
- `GET /v1/conversations/{id}/messages`
- `POST /v1/conversations/{id}/messages`

## Testes

```bash
go test ./...
```

## Gerar Swagger localmente

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs
```

## Atalhos com Make

```bash
make help
make up
make logs
make down
```
