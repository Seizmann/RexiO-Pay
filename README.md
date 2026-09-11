# RexiO Pay

Open-source (AGPL-3.0) SMS-verified payment automation platform for Bangladeshi MFS. Merchants accept bKash and Nagad payments on their own personal, agent, or merchant MFS accounts — verified automatically from the payment SMS, no official MFS API needed.

**How it works:** the merchant's Android phone runs [RexiO Pay Engine], which forwards signed payment SMS to the RexiO Pay backend. The backend matches each SMS against a pending checkout session (TrxID + sender number + time window), marks it paid, and fires a signed webhook to the merchant's site.

A SpritexAI product — [spritexai.pro.bd](https://spritexai.pro.bd) · part of the RexiO family ([rexio.pro](https://rexio.pro)) by Mohammad Sijan.

## Repository layout

```
apps/
  web/        Next.js + TypeScript + Tailwind v4 (merchant dashboard, checkout, admin)
  backend/    Go (chi + pgx + sqlc) — matching engine, parser, webhooks, device API
  android/    RexiO Pay Engine (Kotlin + Jetpack Compose) — SMS receiver app
packages/
  shared-types/   Shared TS types (API contracts, webhook payloads)
  openapi/         OpenAPI 3.1 spec — merchant API source of truth
supabase/     RLS policies and SQL
```

## Getting started

Requires: Go 1.22+, Node 20+, pnpm 9+.

```bash
# backend
cd apps/backend && go run ./cmd/server

# web
pnpm install && pnpm dev
```

See the API reference at `packages/openapi/openapi.yaml` and the deployment notes in `docker-compose.yml`.

## License

AGPL-3.0 — see [LICENSE](LICENSE). RexiO Pay does not hold or move merchant funds; it is a verification and automation layer over payments merchants receive in their own MFS accounts. Each merchant is responsible for their MFS provider's terms.
