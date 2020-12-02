# Rent A Minecraft Server

## Deployment

The project is now live at [https://netheriteblock.com](https://netheriteblock.com).

## Preamble

This was the side project assignment for USFCA CS 601 Fall 2020. However, the project is now open-source.

See the [designs](./designs) folder for how the project was designed.

See the [docs](./docs) folder for documentations.

See the [Spring Boards](https://github.com/miragespace/rmc/projects) for current tasks.

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
- miragespace: globally redundant backend service

Services:
- Mailgun: For outbound SMTP sending passwordless token emails
- Sentry: For error monitoring
- Stripe: For Usage-Based Billing
- Cloudflare: For API endpoint security
