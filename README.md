# Deblock Transaction Monitor

## Overview

Deblock is a robust transaction monitoring service designed to track Ethereum blockchain transactions for specific addresses in real-time. It provides a comprehensive solution for monitoring and processing blockchain events with distributed locking and event publishing capabilities.

## Features

- Real-time Ethereum blockchain transaction monitoring
- Distributed locking mechanism using Redis
- Event publishing via Kafka
- Configurable watched Ethereum addresses
- Docker-based deployment
- RESTful API for controlling transaction monitoring
- Comprehensive logging and error handling

## Prerequisites

- Go 1.21+
- Docker
- Docker Compose
- Git

## Configuration

The application is configured using environment variables, which can be set directly in the `docker-compose.yml` file:

```yaml
environment:
  - SERVER_PORT=8080
  - LOG_LEVEL=info
  - GIN_MODE=release
  - ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
  - ETHEREUM_WS_URL=wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID
  - REDIS_URL=redis://redis:6379
  - KAFKA_BROKERS=kafka:9092
  - WATCHED_ADDRESSES=0x1234567890123456789012345678901234567890,0x0987654321098765432109876543210987654321
```

### Configuration Parameters

- `SERVER_PORT`: Port on which the API server runs
- `LOG_LEVEL`: Logging verbosity (`debug`, `info`, `warn`, `error`)
- `GIN_MODE`: Gin framework mode (`debug`, `release`, `test`)
- `ETHEREUM_RPC_URL`: Ethereum JSON-RPC endpoint
- `ETHEREUM_WS_URL`: Ethereum WebSocket endpoint for real-time block subscriptions
- `REDIS_URL`: Redis connection URL for distributed locking
- `KAFKA_BROKERS`: Kafka broker addresses for event publishing
- `WATCHED_ADDRESSES`: Comma-separated list of Ethereum addresses to monitor

## Running the Application

### Local Development

1. Clone the repository
   ```bash
   git clone https://github.com/ganeshdipdumbare/deblock.git
   cd deblock
   ```

2. Start the services
   ```bash
   docker-compose up --build
   ```

### Services

- **Transaction Monitor API**: `http://localhost:8080`
- **Swagger Documentation**: `http://localhost:8080/api/v1/swagger/index.html`
- **Kafka UI**: `http://localhost:8090`

## API Endpoints

- `POST /api/v1/txmonitor/start`: Start transaction monitoring
- `POST /api/v1/txmonitor/stop`: Stop transaction monitoring
- `GET /api/v1/health`: Check service health
- `GET /api/v1/swagger/*`: Swagger API documentation

## Development

### Running Tests

```bash
# Run unit tests
make test

# Run integration tests
make test-integration
```

### Building the Application

```bash
# Install dependencies
make deps

# Build the application
make build

# Build Docker image
make docker-build
```

## Monitoring and Management

- Use Kafka UI to monitor transaction events
- Inspect application logs for detailed monitoring

## System Resilience and Edge Case Handling

### Blockchain Node Downtime and Transaction Recovery

#### Scenario: Blockchain Node Unavailable for 1 Hour

1. **Block Synchronization Strategy**
   - Implement a multi-layered block recovery mechanism
   - Use multiple Ethereum node providers (Infura, Alchemy, self-hosted)
   - Maintain an in-memory and persistent block checkpoint

2. **Transaction Recovery Process**
   ```
   When node reconnects:
   1. Retrieve last processed block number from persistent storage
   2. Fetch missed blocks using JSON-RPC `eth_getBlockByNumber`
   3. Process blocks in batches to prevent overwhelming the system
   4. Deduplicate transactions using transaction hash
   5. Publish missed transactions to Kafka
   ```

3. **Retry and Backoff Mechanism**
   - Exponential backoff for node reconnection attempts
   - Circuit breaker pattern to prevent continuous reconnection attempts
   - Configurable retry parameters:
     ```go
     type RetryConfig struct {
         BaseDelay   time.Duration  // Initial delay
         MaxDelay    time.Duration  // Maximum delay between retries
         MaxRetries  int            // Maximum number of retry attempts
     }
     ```

### Block Reorganization Handling

1. **Blockchain Reorg Detection**
   - Track block confirmations (typically 6 blocks for Ethereum)
   - Implement a sliding window for block validation
   - Maintain a temporary cache of potentially unstable transactions

2. **Reorg Handling Strategy**
   ```
   When block reorganization detected:
   1. Compare new chain with previous chain
   2. Identify transactions that are no longer valid
   3. Remove invalidated transactions from processing queue
   4. Republish valid transactions from the new chain
   5. Log reorg event for audit purposes
   ```

### Idempotency and Exactly-Once Processing

1. **Transaction Deduplication**
   - Use transaction hash as unique identifier
   - Implement a distributed cache (Redis) to track processed transactions
   - Ensure each transaction is processed exactly once

2. **Kafka Exactly-Once Semantics**
   - Utilize Kafka's transactional producer
   - Implement idempotent producer configuration
   - Use unique transaction IDs for each message

### Failure Scenarios and Mitigation

1. **Network Interruptions**
   - Implement persistent message queues
   - Use Kafka's message retention for replay
   - Configurable message TTL and retry mechanisms

2. **Resource Exhaustion**
   - Implement backpressure mechanisms
   - Use circuit breakers
   - Dynamic scaling of processing workers

3. **Monitoring and Alerting**
   - Comprehensive logging of all critical events
   - Prometheus metrics for system health
   - Alerting on prolonged processing delays or repeated failures

### Performance Considerations

1. **Horizontal Scalability**
   - Stateless design of transaction monitor
   - Kafka as a distributed message queue
   - Ability to run multiple instances with distributed locking

2. **Memory Management**
   - Implement sliding window for block processing
   - Configurable block history retention
   - Periodic cleanup of processed block data

### Security Considerations

1. **Secure Configuration**
   - Use environment-based configuration
   - Support for encrypted configuration sources
   - Minimal permission principle for service accounts

2. **Rate Limiting**
   - Implement rate limiting on Ethereum node requests
   - Configurable request throttling
   - Fallback and failover mechanisms

### Potential Future Enhancements

1. Machine learning-based anomaly detection
2. Support for multiple blockchain networks
3. Advanced filtering and custom transaction processing rules
4. Real-time dashboard for transaction monitoring

## Conclusion

The Deblock Transaction Monitor is designed with resilience, scalability, and reliability as core principles. By implementing comprehensive error handling, retry mechanisms, and intelligent processing strategies, the system ensures robust blockchain transaction tracking even in challenging network and infrastructure conditions.

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Contact

Ganeshdip Dumbare - [GitHub](https://github.com/ganeshdipdumbare)
