# Agent Manager HTTP Server

A comprehensive Go HTTP server implementation with routing, middleware, error handling, and comprehensive testing.

## Architecture

The server follows a clean architecture pattern with clear separation of concerns:

```
cmd/server/          # Application entry point
internal/
├── config/          # Configuration management
├── handlers/        # HTTP request handlers
├── middleware/      # HTTP middleware stack
├── models/          # Data models and validation
├── repository/      # Data access layer (Redis)
└── service/         # Business logic layer
```

## Features

### HTTP Server
- **Graceful Shutdown**: Proper signal handling with 30-second timeout
- **Request Routing**: Modern Go 1.22+ routing with path parameters
- **Structured Logging**: Request ID tracking and detailed logging
- **Health Checks**: `/health` and `/health/ready` endpoints
- **CORS Support**: Cross-origin resource sharing middleware

### Middleware Stack
- **Request ID**: Unique identifier for request tracing
- **Logging**: Structured HTTP request/response logging  
- **CORS**: Cross-origin resource sharing
- **Recovery**: Panic recovery with stack traces
- **Timeout**: Configurable request timeouts (30s default)
- **Rate Limiting**: In-memory rate limiter (100 req/min default)

### API Endpoints

#### Handoff Management
```
POST   /api/v1/handoffs              # Create new handoff
GET    /api/v1/handoffs/{id}         # Get handoff by ID
GET    /api/v1/handoffs              # List handoffs (with pagination)
PUT    /api/v1/handoffs/{id}/status  # Update handoff status
```

#### Queue Management
```
GET    /api/v1/queues                # List all queues
GET    /api/v1/queues/{queue}/depth  # Get queue depth
```

#### Health Checks
```
GET    /health                       # Basic health check
GET    /health/ready                 # Readiness check (includes Redis)
```

### Error Handling
- **Structured Errors**: Consistent JSON error responses
- **Request ID Tracking**: Error correlation across logs
- **Validation Errors**: Clear validation failure messages
- **Status Code Mapping**: Proper HTTP status codes

### Data Models
- **Handoff**: Core task handoff between agents
- **Priority Levels**: Low, Normal, High, Urgent with queue scoring
- **Status Transitions**: Pending → Processing → Completed/Failed/Cancelled
- **Validation**: Input validation with descriptive error messages

## Configuration

Environment variables with sensible defaults:

```bash
# Server Configuration
SERVER_ADDRESS=:8080                    # Server bind address
SERVER_READ_TIMEOUT=10s                 # Request read timeout
SERVER_WRITE_TIMEOUT=10s                # Response write timeout
SERVER_IDLE_TIMEOUT=60s                 # Connection idle timeout

# Redis Configuration  
REDIS_ADDR=localhost:6379               # Redis server address
REDIS_PASSWORD=                         # Redis password (optional)
REDIS_DB=0                             # Redis database number

# Environment
ENV=development                         # Environment (development/production)
```

## Building and Running

### Build
```bash
make server                             # Build HTTP server binary
make manager                            # Build existing manager
make publisher                          # Build existing publisher
make build                              # Build all binaries
```

### Run
```bash
make run-server                         # Build and run HTTP server
make run-manager                        # Build and run existing manager  
```

### Test
```bash
make test                              # Run all tests with coverage
make test-verbose                      # Run tests with verbose output
make bench                             # Run benchmarks
```

### Development
```bash
make lint                              # Lint code (go vet + gofmt)
make fmt                               # Format code
make dev                               # Full dev cycle (clean, lint, test, build)
```

## Docker Support

### Build Images
```bash
make docker-build                      # Build Docker image
docker-compose up -d                   # Run with Redis + monitoring
```

### Production Deployment
```bash
make prod-build                        # Optimized production builds
docker-compose --profile monitoring up # Include Prometheus/Grafana
```

## Testing

### Test Coverage
- **Handlers**: 44.5% coverage with comprehensive unit tests
- **Mocking**: Interface-based mocking for clean test isolation
- **HTTP Testing**: `httptest` for handler testing
- **Table-Driven**: Structured test cases for multiple scenarios

### Test Examples
```go
func TestHandoffHandler_CreateHandoff(t *testing.T) {
    tests := []struct {
        name           string
        payload        models.CreateHandoffRequest
        expectedStatus int
    }{
        {
            name: "valid handoff creation",
            payload: models.CreateHandoffRequest{
                ProjectName: "test-project",
                Summary:     "Test handoff",
                // ...
            },
            expectedStatus: http.StatusCreated,
        },
    }
    // ... test implementation
}
```

## Performance Features

### Connection Management
- **Connection Pooling**: Redis connection pool with limits
- **Keep-Alive**: HTTP keep-alive connections
- **Timeouts**: Configurable timeouts at all levels

### Memory Management
- **Object Pooling**: sync.Pool for buffer reuse
- **Streaming**: JSON streaming for large responses
- **Rate Limiting**: Memory-efficient rate limiting

### Monitoring Ready
- **Metrics**: Prometheus metrics integration ready
- **Health Checks**: Deep health checks with Redis connectivity
- **Structured Logs**: JSON logs for aggregation

## Security Features

### Input Validation
- **Request Validation**: Comprehensive input validation
- **Size Limits**: Request size limits
- **SQL Injection Prevention**: Parameterized queries (Redis commands)

### Headers and CORS  
- **Security Headers**: Request ID, CORS headers
- **CORS Configuration**: Configurable CORS policies
- **Request Limits**: Rate limiting per client IP

## Production Ready

### Observability
- **Request Tracing**: Request ID throughout request lifecycle  
- **Structured Logging**: JSON logs with context
- **Health Monitoring**: Multiple health check endpoints

### Reliability
- **Graceful Shutdown**: Clean shutdown handling
- **Panic Recovery**: Panic recovery with logging
- **Circuit Breaker Ready**: Interface ready for circuit breaker patterns

### Performance
- **Optimized Builds**: Production builds with size optimization
- **Memory Efficient**: Minimal memory footprint
- **Concurrent Safe**: Thread-safe operations throughout

## API Examples

### Create Handoff
```bash
curl -X POST http://localhost:8080/api/v1/handoffs \
  -H "Content-Type: application/json" \
  -d '{
    "project_name": "agent-manager",
    "from_agent": "golang-expert", 
    "to_agent": "test-expert",
    "summary": "HTTP server implementation complete",
    "priority": "normal"
  }'
```

### Get Handoff
```bash
curl http://localhost:8080/api/v1/handoffs/{handoff-id}
```

### List Handoffs
```bash
curl "http://localhost:8080/api/v1/handoffs?project=agent-manager&page=1&page_size=20"
```

### Health Check
```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/ready
```

## Next Steps

The HTTP server implementation is complete and ready for comprehensive testing including:

- **Unit Tests**: HTTP handler testing with mocks
- **Integration Tests**: End-to-end API testing with Redis
- **Load Testing**: Performance testing under load
- **Security Testing**: Input validation and security scanning

The implementation demonstrates Go best practices with:
- Clean architecture
- Comprehensive error handling  
- Extensive middleware stack
- Production-ready configuration
- Docker containerization
- Monitoring and observability features