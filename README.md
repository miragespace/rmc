# Rent A Minecraft Server

## Preamble

This is the side project assignment for USFCA CS 601 Fall 2020.

See the [designs](./designs) folder for how the project was designed.
o

## Tech Stack for the Project

Backend:
- Golang 1.15.3
- Protobuf
- GitHub Actions (CI coming soon)

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

## TODO

1. ~~Structual logging with zap~~

2. Add hooks to manage instances

3. Refactor error handling into a package

4. Make all response JSON

5. Add tests to all packages

```
(c) 2020 Rachel Chen
```