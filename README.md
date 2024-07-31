# Mahogany

> Break it down? Are you kidding me? This is hand-carved mahogany!

_A dead-simple web UI for managing docker containers._

## Getting Started
I don't trust random docker images published on the internet and neither should you. Build and run the docker image yourself.

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
docker build -t mahogany
docker run --restart=always --name mahogany -p 9090:9090 mahogany
```

In addition to managing docker containers, mahogany also integrates with [registry](https://hub.docker.com/_/registry) and [watchtower](https://containrrr.dev/watchtower/). To start everything together, use `docker compose up`!
