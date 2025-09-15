package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vot3k/agent-handoff/handoff"
)

var (
	configFile = flag.String("config", "config.json", "Configuration file path")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	redisAddr  = flag.String("redis-addr", "localhost:6379", "Redis server address")
	redisDB    = flag.Int("redis-db", 0, "Redis database number")
)

// ServiceConfig represents the complete service configuration
type ServiceConfig struct {
	Redis struct {
		Addr     string `json:"addr"`
		Password string `json:"password,omitempty"`
		DB       int    `json:"db"`
	} `json:"redis"`

	Logging struct {
		Level string `json:"level"`
	} `json:"logging"`

	Agents []handoff.AgentCapabilities `json:"agents"`

	Routes map[string][]handoff.RouteRule `json:"routes"`

	AlertRules []handoff.AlertRule `json:"alert_rules"`

	Monitoring struct {
		Enabled  bool          `json:"enabled"`
		Interval time.Duration `json:"interval"`
	} `json:"monitoring"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() ServiceConfig {
	return ServiceConfig{
		Redis: struct {
			Addr     string `json:"addr"`
			Password string `json:"password,omitempty"`
			DB       int    `json:"db"`
		}{
			Addr: "localhost:6379",
			DB:   0,
		},
		Logging: struct {
			Level string `json:"level"`
		}{
			Level: "info",
		},
		Agents: []handoff.AgentCapabilities{
			{
				Name:          "api-expert",
				Description:   "API design and specification expert",
				Triggers:      []string{"api", "endpoint", "swagger", "openapi"},
				InputTypes:    []string{"requirements", "specifications"},
				OutputTypes:   []string{"api-spec", "openapi-doc"},
				QueueName:     "handoff:queue:api-expert",
				MaxConcurrent: 5,
			},
			{
				Name:          "golang-expert",
				Description:   "Go/Golang implementation expert",
				Triggers:      []string{"implement", "go", "golang", "backend"},
				InputTypes:    []string{"api-spec", "requirements"},
				OutputTypes:   []string{"go-code", "implementation-summary"},
				QueueName:     "handoff:queue:golang-expert",
				MaxConcurrent: 3,
			},
			{
				Name:          "typescript-expert",
				Description:   "TypeScript/React implementation expert",
				Triggers:      []string{"frontend", "typescript", "react", "ui"},
				InputTypes:    []string{"design-spec", "api-spec"},
				OutputTypes:   []string{"typescript-code", "react-components"},
				QueueName:     "handoff:queue:typescript-expert",
				MaxConcurrent: 3,
			},
			{
				Name:          "test-expert",
				Description:   "Testing and quality assurance expert",
				Triggers:      []string{"test", "coverage", "qa", "quality"},
				InputTypes:    []string{"implementation", "code"},
				OutputTypes:   []string{"test-code", "coverage-report"},
				QueueName:     "handoff:queue:test-expert",
				MaxConcurrent: 2,
			},
			{
				Name:          "devops-expert",
				Description:   "DevOps and deployment expert",
				Triggers:      []string{"deploy", "docker", "kubernetes", "ci/cd"},
				InputTypes:    []string{"implementation", "requirements"},
				OutputTypes:   []string{"deployment-config", "ci-config"},
				QueueName:     "handoff:queue:devops-expert",
				MaxConcurrent: 2,
			},
		},
		Routes: map[string][]handoff.RouteRule{
			"api-expert": {
				{
					Name:        "route-go-implementation",
					TargetAgent: "golang-expert",
					Priority:    100,
					Conditions: []handoff.RouteCondition{
						{
							Type:          handoff.ConditionComplexQuery,
							Field:         "has_go_files",
							Operator:      "equals",
							Value:         true,
							CaseSensitive: false,
						},
					},
				},
				{
					Name:        "route-typescript-implementation",
					TargetAgent: "typescript-expert",
					Priority:    90,
					Conditions: []handoff.RouteCondition{
						{
							Type:          handoff.ConditionComplexQuery,
							Field:         "has_typescript_files",
							Operator:      "equals",
							Value:         true,
							CaseSensitive: false,
						},
					},
				},
			},
			"golang-expert": {
				{
					Name:        "route-to-testing",
					TargetAgent: "test-expert",
					Priority:    100,
					Conditions: []handoff.RouteCondition{
						{
							Type:          handoff.ConditionComplexQuery,
							Field:         "is_implementation_handoff",
							Operator:      "equals",
							Value:         true,
							CaseSensitive: false,
						},
					},
				},
			},
			"test-expert": {
				{
					Name:        "route-to-deployment",
					TargetAgent: "devops-expert",
					Priority:    100,
					Conditions: []handoff.RouteCondition{
						{
							Type:     handoff.ConditionContent,
							Field:    "summary",
							Operator: "contains",
							Value:    "deploy",
						},
					},
				},
			},
		},
		AlertRules: []handoff.AlertRule{
			{
				Name:      "high-queue-depth",
				Type:      handoff.AlertQueueDepth,
				Condition: "greater_than",
				Threshold: 50,
				Duration:  time.Minute,
				Enabled:   true,
				Cooldown:  5 * time.Minute,
			},
			{
				Name:      "high-failure-rate",
				Type:      handoff.AlertFailureRate,
				Condition: "greater_than",
				Threshold: 10.0,
				Duration:  5 * time.Minute,
				Enabled:   true,
				Cooldown:  10 * time.Minute,
			},
			{
				Name:      "slow-processing",
				Type:      handoff.AlertProcessingTime,
				Condition: "greater_than",
				Threshold: 30000, // 30 seconds in milliseconds
				Duration:  2 * time.Minute,
				Enabled:   true,
				Cooldown:  5 * time.Minute,
			},
			{
				Name:      "system-health-low",
				Type:      handoff.AlertSystemHealth,
				Condition: "less_than",
				Threshold: 50,
				Duration:  time.Minute,
				Enabled:   true,
				Cooldown:  10 * time.Minute,
			},
		},
		Monitoring: struct {
			Enabled  bool          `json:"enabled"`
			Interval time.Duration `json:"interval"`
		}{
			Enabled:  true,
			Interval: 30 * time.Second,
		},
	}
}

func main() {
	flag.Parse()

	// Setup logging
	level, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid log level")
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	config := loadConfig(*configFile)

	// Override with command line flags
	if *redisAddr != "localhost:6379" {
		config.Redis.Addr = *redisAddr
	}
	if *redisDB != 0 {
		config.Redis.DB = *redisDB
	}
	if *logLevel != "info" {
		config.Logging.Level = *logLevel
	}

	// Create optimized handoff agent
	poolConfig := handoff.DefaultRedisPoolConfig()
	poolConfig.Addr = config.Redis.Addr
	poolConfig.Password = config.Redis.Password
	poolConfig.DB = config.Redis.DB

	handoffConfig := handoff.OptimizedConfig{
		RedisConfig: poolConfig,
		LogLevel:    config.Logging.Level,
	}

	agent, err := handoff.NewOptimizedHandoffAgent(handoffConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create handoff agent")
	}
	defer agent.Close()

	// Register agents
	for _, agentCap := range config.Agents {
		if err := agent.RegisterAgent(agentCap); err != nil {
			log.Fatal().Err(err).Str("agent", agentCap.Name).Msg("Failed to register agent")
		}
		log.Info().
			Str("agent", agentCap.Name).
			Str("queue", agentCap.QueueName).
			Int("max_concurrent", agentCap.MaxConcurrent).
			Msg("Agent registered")
	}

	// Setup router
	router := handoff.NewHandoffRouter("default-agent")
	for fromAgent, rules := range config.Routes {
		for _, rule := range rules {
			router.AddRoute(fromAgent, rule)
			log.Info().
				Str("from_agent", fromAgent).
				Str("rule_name", rule.Name).
				Str("target_agent", rule.TargetAgent).
				Int("priority", rule.Priority).
				Msg("Route rule added")
		}
	}

	// Setup monitoring
	var monitor *handoff.OptimizedHandoffMonitor
	if config.Monitoring.Enabled {
		monitor = handoff.NewOptimizedHandoffMonitor(agent.GetRedisManager())

		// Add alert rules
		for _, rule := range config.AlertRules {
			monitor.AddAlertRule(rule)
		}

		// Subscribe to alerts and log them
		alertChan := monitor.SubscribeToAlerts("all")
		go func() {
			for alert := range alertChan {
				log.Warn().
					Str("rule", alert.Rule.Name).
					Float64("value", alert.Value).
					Str("severity", string(alert.Severity)).
					Str("message", alert.Message).
					Msg("Alert triggered")
			}
		}()

		// Start monitoring
		go func() {
			ctx := context.Background()
			monitor.StartMonitoring(ctx, config.Monitoring.Interval)
		}()

		log.Info().
			Dur("interval", config.Monitoring.Interval).
			Int("alert_rules", len(config.AlertRules)).
			Msg("Monitoring started")
	}

	// Setup example consumer (this would be replaced by actual agent implementations)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a demo consumer for golang-expert
	go func() {
		log.Info().Msg("Starting demo golang-expert consumer")
		err := agent.ConsumeHandoffs(ctx, "golang-expert", func(ctx context.Context, h *handoff.Handoff) error {
			log.Info().
				Str("handoff_id", h.Metadata.HandoffID).
				Str("from_agent", h.Metadata.FromAgent).
				Str("summary", h.Content.Summary).
				Msg("Processing handoff")

			// Simulate processing time
			time.Sleep(time.Duration(100+time.Now().UnixNano()%1000) * time.Millisecond)

			// Simulate occasional failures (10% chance)
			if time.Now().UnixNano()%10 == 0 {
				return fmt.Errorf("simulated processing error")
			}

			log.Info().
				Str("handoff_id", h.Metadata.HandoffID).
				Msg("Handoff processed successfully")

			return nil
		})

		if err != nil {
			log.Error().Err(err).Msg("Consumer error")
		}
	}()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info().Msg("Handoff agent service started")
	log.Info().Msg("Service is ready to handle handoffs")

	// Wait for shutdown signal
	<-sigChan
	log.Info().Msg("Shutdown signal received")

	// Graceful shutdown
	cancel()

	log.Info().Msg("Handoff agent service stopped")
}

// loadConfig loads configuration from file, falling back to defaults
func loadConfig(filename string) ServiceConfig {
	config := DefaultConfig()

	if filename == "" {
		log.Info().Msg("No config file specified, using defaults")
		return config
	}

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Str("file", filename).Msg("Config file not found, using defaults")
			return config
		}
		log.Fatal().Err(err).Str("file", filename).Msg("Failed to open config file")
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Fatal().Err(err).Str("file", filename).Msg("Failed to parse config file")
	}

	log.Info().Str("file", filename).Msg("Configuration loaded")
	return config
}
