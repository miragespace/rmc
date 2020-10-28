# Deploying RMC

RMC has 3 main component:
1. The API server (`cmd/api`): responding to user requests and publishing controls for instances
2. The background task service (`cmd/task`): updating instance status and billing reporting
3. Minecraft server host worker (`cmd/worker`): receiving requests to control/provision instances

Run `make all` to build the binaries for these three components, then use your favorite deployment tool (on-premise Kubernetes, GKE, etc).

Once you have your `.env` file ready, rename it `.env.production`, and deploy each component with `ENV=production`.

Network & Access:
1. The API server will need both `.env.production` and `plans.json`, and it requires access to all the external dependencies (e.g. PostgreSQL), and responds to API requests.
2. The background task service needs `.env.production` and `plans.json`, and it only require access to PostgreSQL, AMQP, and Stripe. It will not accept requests from users.
3. Host worker runs on all of your servers that will provision Minecraft server, and it only require access to AMQP. Once it successful starts for the first time, it will automatically register itself with the API server.

(TODO: random ports)
(TODO: security)