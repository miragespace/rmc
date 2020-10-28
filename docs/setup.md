# Setting Up Your Own "Rent a Minecraft Server"

## Technology Stack

RMC uses the following data stores:
1. PostgreSQL (`POSTGRES_URL`)
2. RabbitMQ (`AMQP_URI`) or nats-server (coming soon)
3. Redis (`REDIS_URI` and `REDIS_PW`)
4. Set a long JWT key (`JWT_KEY`). You can run `openssl rand -hex 64` to quickly generate one

Please have those setup and configure your environment accordingly. See `.env.example` for the variables needed.

## Services

RMC uses the following external services:
1. Mailgun (or any Transactional Email API Services that provides SMTP) (`SMTP_*`)
2. Stripe (how else are you getting paid) (`STRIPE_KEY`)
3. (Recommended) Sentry for error reporting. This allows easier operations. (`SENTRY_DSN`)

See `.env.example` for the variables needed.

## Price List

See `plans.json` for example. Note that this file is required for subscription service, and the API server will refuse to start if it cannot find or parse the master price list JSON.

*Note*:
1. Treat this as an append-only list. Once the API server starts and synchronize Plans and Parts with Stripe, making changes for the existing items will cause the API server to misbehave.
2. If you need to make changes to an existing plan, make a new plan under a **different** name, then adjust the new plan accordingly, and mark the old plan as Retired (`"retired": true`).
3. `parameters` under each Plan will be used when provisioning new Minecraft server instances, and you *can* change it after your `plans.json` has been synchronized with Stripe. However, it will only apply to new Instances and will not apply retroactively.

## Endpoint

(TODO)


## Once you have everything, See `deployment.md`