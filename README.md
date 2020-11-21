# Rent A Minecraft Server

## Deployment

It is now live at [https://netheriteblock.com](https://netheriteblock.com). The branding will be changed soon/

## Preamble

This is the side project assignment for USFCA CS 601 Fall 2020.

See the [designs](./designs) folder for how the project was designed.

See the [docs](./docs) folder for documentations.

See the [Spring Boards](https://github.com/zllovesuki/rmc/projects) for current tasks.

## Tech Stack for the Project

Backend:
- Golang 1.15.3
- Protobuf
- GitHub Actions (CI coming soon)

Frontend:
- Vue.js
- Cloudflare Workers Sites

Data:
- Redis: For passwordless login token storage
- PostgreSQL: For persistence data storage & usage accounting
- RabbitMQ: Message broker for controlling Minecraft servers

Infrastructure:
- Redislab: For managed Redis hosting
- CloudAMQP: For managed RabbitMQ hosting
- ElephantSQL: For managed PostgreSQL hosting
- Docker: For local development and containers for Minecraft servers

Services:
- Mailgun: For outbound SMTP sending passwordless token emails
- Sentry: For error monitoring
- Stripe: For Usage-Based Billing
- Cloudflare: For API endpoint security

## TODO

1. ~~Structual logging with zap~~

2. ~~Add hooks to manage instances~~ Dependency injection

3. ~~Refactor error handling into a package~~ WIP

4. ~~Make all response JSON~~ WIP

5. Add tests to all packages

```
(c) 2020 Rachel Chen
```
