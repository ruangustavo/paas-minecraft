# Minecraft PaaS Platform

Deploy Minecraft servers instantly with custom subdomains. No port management, no complex setup—just create a server and play.

## What This Platform Does

This platform lets you spin up Minecraft servers on-demand using Docker containers. Each server gets its own subdomain (like `server1.ruangustavo.com`) and routes through a single port using infrared reverse proxy. No more managing multiple ports or complex networking.

## How It Works

### Architecture Overview

- **API Server**: Go-based REST API that manages server lifecycle
- **Docker Containers**: Each Minecraft server runs in an isolated container using `itzg/minecraft-server`
- **Reverse Proxy**: Infrared handles subdomain routing to avoid port conflicts
- **Database**: PostgreSQL stores server metadata and configurations

## Technology Stack

- **Backend**: Go with Echo framework
- **Container Runtime**: Docker with official Minecraft server image
- **Database**: PostgreSQL with GORM
- **Reverse Proxy**: Infrared for subdomain routing
- **Frontend**: React (planned)

## Future Improvements

### Horizontal Scaling

The current single-machine setup works for learning and small deployments. Future versions will support:

- **Cloud Integration**: Deploy containers across multiple cloud instances
- **Load Balancing**: Distribute servers across multiple hosts
- **Auto-scaling**: Spin up/down servers based on demand
- **Multi-region**: Deploy servers closer to players

## Why This Approach?

This project explores several important concepts:

- **Container Orchestration**: Managing Docker containers programmatically
- **Reverse Proxy Patterns**: Routing traffic without port conflicts
- **Database Design**: Tracking container state and metadata
- **Go Concurrency**: Handling multiple server operations efficiently

The single-machine approach makes it perfect for learning these concepts without cloud complexity. You can run everything locally and understand how each piece works together.

## License

MIT License - feel free to use this code for your own Minecraft hosting experiments.
