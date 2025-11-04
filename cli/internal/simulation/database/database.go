package database

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/lib/pq"
	"github.com/nitrictech/suga/cli/internal/netx"
	"github.com/nitrictech/suga/cli/internal/style"
	"github.com/nitrictech/suga/cli/pkg/schema"
)

const (
	postgresImage    = "ghcr.io/fboulnois/pg_uuidv7:1.6.0"
	postgresUser     = "suga"
	postgresPassword = "suga"
	postgresDB       = "postgres"
)

// DatabaseManager manages a single PostgreSQL Docker container with multiple databases
type DatabaseManager struct {
	dockerClient *client.Client
	containerID  string
	volumeName   string
	port         netx.ReservedPort
	databases    map[string]*DatabaseInfo
	ctx          context.Context
}

// DatabaseInfo holds information about a specific database
type DatabaseInfo struct {
	name   string
	intent schema.DatabaseIntent
}

// NewDatabaseManager creates a new database manager
func NewDatabaseManager(projectName, dataDir string) (*DatabaseManager, error) {
	ctx := context.Background()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	port, err := netx.ReservePort(5432)
	if err != nil {
		port, err = netx.GetNextPort(netx.MinPort(5433), netx.MaxPort(6000))
		if err != nil {
			return nil, fmt.Errorf("failed to allocate port for PostgreSQL: %w", err)
		}
	}

	sanitizedName := sanitizeVolumeName(projectName)
	volumeName := fmt.Sprintf("suga-%s-postgres-data", sanitizedName)

	return &DatabaseManager{
		dockerClient: dockerClient,
		volumeName:   volumeName,
		port:         port,
		databases:    make(map[string]*DatabaseInfo),
		ctx:          ctx,
	}, nil
}

// Start initializes and starts the PostgreSQL Docker container
func (m *DatabaseManager) Start() error {
	_, err := m.dockerClient.VolumeInspect(m.ctx, m.volumeName)
	if err != nil {
		_, err = m.dockerClient.VolumeCreate(m.ctx, volume.CreateOptions{
			Name: m.volumeName,
			Labels: map[string]string{
				"created-by": "suga",
				"purpose":    "postgres-data",
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create Docker volume: %w", err)
		}
	}

	fmt.Printf("Pulling PostgreSQL image %s (this may take some time on first run)...\n\n",
		style.Cyan(postgresImage))
	reader, err := m.dockerClient.ImagePull(m.ctx, postgresImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull PostgreSQL image: %w", err)
	}
	defer reader.Close()

	// Wait for image pull to complete
	_, _ = io.Copy(io.Discard, reader)

	containerConfig := &container.Config{
		Image: postgresImage,
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", postgresUser),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", postgresPassword),
			fmt.Sprintf("POSTGRES_DB=%s", postgresDB),
		},
		ExposedPorts: nat.PortSet{
			"5432/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"5432/tcp": []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: fmt.Sprintf("%d", m.port),
				},
			},
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: m.volumeName,
				Target: "/var/lib/postgresql/data",
			},
		},
		AutoRemove: true,
	}

	resp, err := m.dockerClient.ContainerCreate(
		m.ctx,
		containerConfig,
		hostConfig,
		nil,
		nil,
		"",
	)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL container: %w", err)
	}

	m.containerID = resp.ID

	if err := m.dockerClient.ContainerStart(m.ctx, m.containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	if err := m.waitForPostgres(); err != nil {
		return fmt.Errorf("PostgreSQL failed to become ready: %w", err)
	}

	return nil
}

// waitForPostgres waits for PostgreSQL to be ready to accept connections
func (m *DatabaseManager) waitForPostgres() error {
	connStr := fmt.Sprintf("host=localhost port=%d user=%s password=%s dbname=%s sslmode=disable",
		m.port, postgresUser, postgresPassword, postgresDB)

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		err = db.Ping()
		db.Close()

		if err == nil {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("PostgreSQL did not become ready within timeout")
}

// CreateDatabase creates a new database in the PostgreSQL instance if it doesn't exist
func (m *DatabaseManager) CreateDatabase(name string, intent schema.DatabaseIntent) error {
	connStr := fmt.Sprintf("host=localhost port=%d user=%s password=%s dbname=%s sslmode=disable",
		m.port, postgresUser, postgresPassword, postgresDB)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	// Try to create the database
	// PostgreSQL doesn't support CREATE DATABASE IF NOT EXISTS, so we catch the error
	// Use pq.QuoteIdentifier to safely handle database names with special characters
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", pq.QuoteIdentifier(name)))
	if err != nil {
		// Check if error is "database already exists" (error code 42P04)
		// If so, that's fine - we can ignore it
		if !isDatabaseExistsError(err) {
			return fmt.Errorf("failed to create database %s: %w", name, err)
		}
		// Database already exists, continue
	}

	m.databases[name] = &DatabaseInfo{
		name:   name,
		intent: intent,
	}

	return nil
}

// isDatabaseExistsError checks if the error is a "database already exists" error
func isDatabaseExistsError(err error) bool {
	// PostgreSQL error code 42P04 is "duplicate_database"
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "42P04"
	}
	return false
}

// Stop shuts down and removes the PostgreSQL container
func (m *DatabaseManager) Stop() error {
	if m.containerID == "" {
		return nil
	}

	timeout := 10
	return m.dockerClient.ContainerStop(m.ctx, m.containerID, container.StopOptions{
		Timeout: &timeout,
	})
}

// RemoveVolume removes the Docker volume (data will be lost)
// This is useful for cleaning up or resetting the database
func (m *DatabaseManager) RemoveVolume() error {
	return m.dockerClient.VolumeRemove(m.ctx, m.volumeName, true)
}

// GetConnectionString returns the PostgreSQL connection string for a specific database
func (m *DatabaseManager) GetConnectionString(name string) string {
	return fmt.Sprintf("postgresql://%s:%s@localhost:%d/%s?sslmode=disable",
		postgresUser, postgresPassword, m.port, name)
}

// GetPort returns the PostgreSQL port number
func (m *DatabaseManager) GetPort() int {
	return int(m.port)
}

// GetEnvVarKey returns the environment variable key for a database
func (m *DatabaseManager) GetEnvVarKey(name string) string {
	if info, ok := m.databases[name]; ok {
		return info.intent.EnvVarKey
	}
	return ""
}

// GetDatabases returns all database names
func (m *DatabaseManager) GetDatabases() []string {
	names := make([]string, 0, len(m.databases))
	for name := range m.databases {
		names = append(names, name)
	}
	return names
}

// sanitizeVolumeName sanitizes a project name for use in Docker volume names
// Docker volume names must match: [a-zA-Z0-9][a-zA-Z0-9_.-]+
func sanitizeVolumeName(name string) string {
	result := strings.ToLower(name)

	invalidChars := regexp.MustCompile(`[^a-z0-9_.-]`)
	result = invalidChars.ReplaceAllString(result, "-")

	if len(result) > 0 && (result[0] == '-' || result[0] == '.') {
		result = "app" + result
	}

	if result == "" {
		result = "app"
	}

	return result
}
