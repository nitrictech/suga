# Plugin Library Development Guide

This guide covers how to create and maintain Suga plugin libraries, which provide the building blocks that platforms compose together.

## Overview

A **plugin library** is a collection of reusable Terraform modules that implement infrastructure components. Each plugin in the library is a self-contained unit that:
- Implements a specific infrastructure resource (e.g., Lambda function, S3 bucket, VPC)
- Defines inputs it requires and outputs it provides
- Contains Terraform code in HCL
- Includes a manifest file describing its interface
- May include runtime code in Go (required for services and buckets)

**Key Concept**: Plugins are the lowest-level building blocks. Platforms compose plugins together, and applications consume platforms.

## Plugin Library Structure

### Repository Layout

```
plugins-aws/                 # Plugin library repository
├── go.mod                   # Go module file for entire library
├── .tflint.hcl              # Terraform linting rules
├── Makefile                 # Build automation
├── lambda/                  # Service plugin
│   ├── manifest.yaml        # Plugin interface definition
│   ├── icon.svg             # UI icon (optional)
│   ├── module/              # Terraform implementation
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   ├── outputs.tf
│   │   └── versions.tf
│   └── runtime.go           # runtime code for translating AWS Lambda events to HTTP
├── s3-bucket/               # Bucket plugin
│   ├── manifest.yaml
│   ├── module/
│   └── runtime.go           # runtime code for interacting with s3 buckets
└── vpc/                     # Infrastructure plugin
    ├── manifest.yaml
    └── module/              # No runtime code needed
```

### Plugin Categories

Plugins typically fall into these categories:

1. **Resource Plugins** - Application-facing resources (services, databases, buckets, entrypoints)
2. **Infrastructure Plugins** - Shared infrastructure (VPCs, load balancers, networking)
3. **Identity Plugins** - IAM roles, service accounts, authentication

## Runtime Code Requirements

### What Is Runtime Code?

Runtime code is **Go code that gets embedded into your Suga application** to facilitate communication between your app and the deployed cloud infrastructure. This is NOT an SDK or library - it is **necessary runtime code** that your application requires to function.

**Currently, runtime code is ONLY written in Go.**

### When Runtime Code Is Required

- **Services** - MUST have runtime code to handle HTTP requests, environment configuration, etc.
- **Buckets** - MUST have runtime code to read/write objects, handle authentication, etc.

### When Runtime Code Is NOT Needed

- **Infrastructure plugins** (VPCs, load balancers) - Pure Terraform, no application interaction
- **Identity plugins** (IAM roles) - Pure Terraform, no application interaction
- **Databases** - Typically use standard database drivers, not plugin-specific runtime code

### Runtime Code Requirements

#### Service Plugin Interface

Service plugins MUST implement the `Service` interface from `github.com/nitrictech/suga/runtime/service`:

```go
type Service interface {
    Start(Proxy) error
}

type Proxy interface {
    Forward(ctx context.Context, req *http.Request) (*http.Response, error)
    Host() string
}
```

**Key Concepts:**
- The `Start` method receives a `Proxy` that forwards HTTP requests to the user's application
- Your runtime code translates cloud-specific events (Lambda events, container requests, etc.) into standard HTTP requests
- The proxy handles forwarding these requests to the user's application running locally
- Your runtime code must register itself using `service.Register()`

Example service runtime code for AWS Lambda:

```go
package lambdaruntime

import (
    "context"
    "net/http"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/nitrictech/suga/runtime/service"
)

type LambdaService struct{}

func (l *LambdaService) Start(proxy service.Proxy) error {
    // Start Lambda runtime with a handler that converts Lambda events to HTTP
    lambda.Start(func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
        // Convert Lambda event to http.Request
        req, err := convertEventToRequest(ctx, event, proxy.Host())
        if err != nil {
            return events.APIGatewayProxyResponse{StatusCode: 500}, err
        }

        // Forward to user's application via proxy
        resp, err := proxy.Forward(ctx, req)
        if err != nil {
            return events.APIGatewayProxyResponse{StatusCode: 500}, err
        }

        // Convert http.Response back to Lambda response
        return convertResponseToEvent(resp)
    })

    return nil
}

func New() (service.Service, error) {
    return &LambdaService{}, nil
}

func init() {
    // Register this service plugin
    service.Register(New)
}
```

#### Bucket Plugin Interface

Bucket plugins MUST implement the `Storage` interface (alias for `StorageServer`) from `github.com/nitrictech/suga/proto/storage/v2`:

```go
type Storage interface {
    // Retrieve an item from a bucket
    Read(context.Context, *StorageReadRequest) (*StorageReadResponse, error)

    // Store an item to a bucket
    Write(context.Context, *StorageWriteRequest) (*StorageWriteResponse, error)

    // Delete an item from a bucket
    Delete(context.Context, *StorageDeleteRequest) (*StorageDeleteResponse, error)

    // Generate a pre-signed URL for direct operations on an item
    PreSignUrl(context.Context, *StoragePreSignUrlRequest) (*StoragePreSignUrlResponse, error)

    // List blobs currently in the bucket
    ListBlobs(context.Context, *StorageListBlobsRequest) (*StorageListBlobsResponse, error)

    // Determine if an object exists in a bucket
    Exists(context.Context, *StorageExistsRequest) (*StorageExistsResponse, error)
}
```

**Key Concepts:**
- Bucket plugins provide object storage operations (read, write, delete, list, etc.)
- The interface uses gRPC protobuf messages for request/response
- Your runtime code must register itself with a namespace using `storage.Register()`
- The namespace typically follows the pattern `<team>/<library>/<plugin-name>`

Example bucket runtime code for AWS S3:

```go
package s3runtime

import (
    "context"
    "io"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    storagepb "github.com/nitrictech/suga/proto/storage/v2"
    "github.com/nitrictech/suga/runtime/storage"
)

type S3StorageService struct {
    storagepb.UnimplementedStorageServer
    s3Client   *s3.Client
    bucketName string
}

func (s *S3StorageService) Read(ctx context.Context, req *storagepb.StorageReadRequest) (*storagepb.StorageReadResponse, error) {
    result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: &s.bucketName,
        Key:    &req.Key,
    })
    if err != nil {
        return nil, err
    }
    defer result.Body.Close()

    body, err := io.ReadAll(result.Body)
    if err != nil {
        return nil, err
    }

    return &storagepb.StorageReadResponse{
        Body: body,
    }, nil
}

func (s *S3StorageService) Write(ctx context.Context, req *storagepb.StorageWriteRequest) (*storagepb.StorageWriteResponse, error) {
    _, err := s.s3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: &s.bucketName,
        Key:    &req.Key,
        Body:   bytes.NewReader(req.Body),
    })
    if err != nil {
        return nil, err
    }

    return &storagepb.StorageWriteResponse{}, nil
}

func (s *S3StorageService) Delete(ctx context.Context, req *storagepb.StorageDeleteRequest) (*storagepb.StorageDeleteResponse, error) {
    _, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: &s.bucketName,
        Key:    &req.Key,
    })
    if err != nil {
        return nil, err
    }

    return &storagepb.StorageDeleteResponse{}, nil
}

// Implement remaining methods: PreSignUrl, ListBlobs, Exists...

func New() (storage.Storage, error) {
    bucketName := os.Getenv("SUGA_BUCKET_NAME")
    if bucketName == "" {
        return nil, fmt.Errorf("SUGA_BUCKET_NAME not set")
    }

    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        return nil, err
    }

    return &S3StorageService{
        s3Client:   s3.NewFromConfig(cfg),
        bucketName: bucketName,
    }, nil
}

// Register with namespace (e.g., "suga/aws/s3-bucket")
// storage.Register("suga/aws/s3-bucket", New)
```

### Runtime Code Responsibilities

Runtime code handles:
1. **Authentication** - Obtaining credentials from the environment
2. **Configuration** - Reading infrastructure details from environment variables
3. **Protocol translation** - Converting between app interfaces and cloud APIs
4. **Error handling** - Translating cloud errors to application-friendly formats

Example runtime code for an S3 bucket plugin:

```go
// s3-bucket/runtime/go/bucket.go
package s3runtime

import (
    "context"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps S3 operations for application use
type Client struct {
    bucketName string
    s3Client   *s3.Client
}

// NewClient creates a bucket client from environment
func NewClient() (*Client, error) {
    // Read bucket name from environment (set by Terraform)
    bucketName := os.Getenv("SUGA_BUCKET_NAME")

    // Create AWS client with default credentials
    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        return nil, err
    }

    return &Client{
        bucketName: bucketName,
        s3Client:   s3.NewFromConfig(cfg),
    }, nil
}

// Put uploads an object to the bucket
func (c *Client) Put(ctx context.Context, key string, data []byte) error {
    _, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(c.bucketName),
        Key:    aws.String(key),
        Body:   bytes.NewReader(data),
    })
    return err
}
```

## Plugin Manifest Structure

Every plugin must have a `manifest.yaml` that defines its interface:

```yaml
name: fargate                           # Plugin identifier
description: AWS Fargate container service
type: resource                          # or "infrastructure" or "identity"

# Optional: Required identity plugins (e.g., IAM roles)
required_identities:
  - aws

# Input parameters the plugin accepts
inputs:
  cpu:
    type: number
    description: CPU units for the container
    required: true
    default: 256

  memory:
    type: number
    description: Memory in MB
    required: true
    default: 512

  container_port:
    type: number
    description: Port the container listens on
    required: true

  vpc_id:
    type: string
    description: VPC ID to deploy into
    required: true

  subnets:
    type: array
    description: Subnet IDs for the service
    required: true

# Output values the plugin provides
outputs:
  service_url:
    type: string
    description: URL of the deployed service

  service_arn:
    type: string
    description: ARN of the ECS service

  task_role_arn:
    type: string
    description: IAM role ARN for the task

# Runtime code configuration (REQUIRED for services/buckets, currently Go only)
runtime:
  go:
    package: github.com/nitrictech/plugins-aws/lambda/runtime/go
```

## Terraform Module Implementation

### Standard Module Files

Each plugin's `module/` directory should contain:

**`main.tf`** - Primary resource definitions:
```hcl
resource "aws_ecs_service" "service" {
  name            = var.name
  cluster         = var.cluster_id
  task_definition = aws_ecs_task_definition.task.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = var.subnets
    security_groups = [aws_security_group.service.id]
  }
}
```

**`variables.tf`** - Input variables matching manifest inputs:
```hcl
variable "cpu" {
  type        = number
  description = "CPU units for the container"
}

variable "memory" {
  type        = number
  description = "Memory in MB"
}

variable "container_port" {
  type        = number
  description = "Port the container listens on"
}
```

**`outputs.tf`** - Output values matching manifest outputs:
```hcl
output "service_url" {
  value       = aws_lb.service.dns_name
  description = "URL of the deployed service"
}

output "service_arn" {
  value       = aws_ecs_service.service.id
  description = "ARN of the ECS service"
}
```

**`versions.tf`** - Provider requirements:
```hcl
terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
```

## Development Workflow

### 1. Planning a New Plugin

Before creating a plugin:

1. **Identify the resource** - What infrastructure component are you implementing?
2. **Define the interface** - What inputs does it need? What outputs does it provide?
3. **Determine runtime needs** - Does it need Go runtime code? (services and buckets MUST have it)
4. **Consider dependencies** - Does it need other plugins (VPC, IAM roles)?
5. **Check for reusability** - Is this plugin generic enough for multiple platforms?

### 2. Creating a New Plugin

```bash
# Create plugin directory structure
mkdir -p my-plugin/module

# For services/buckets, MUST create runtime directory with Go code
mkdir -p my-plugin/runtime/go

# Create manifest
cat > my-plugin/manifest.yaml <<EOF
name: my-plugin
description: Description of what this plugin does
type: resource
inputs: {}
outputs: {}
EOF

# Create Terraform files
touch my-plugin/module/main.tf
touch my-plugin/module/variables.tf
touch my-plugin/module/outputs.tf
touch my-plugin/module/versions.tf

# For services/buckets, create Go runtime files
touch my-plugin/runtime/go/client.go
```

### 3. Implementing the Plugin

1. **Write manifest.yaml** - Define inputs, outputs, and runtime configuration
2. **Implement Terraform** - Write the infrastructure code
3. **Implement Go runtime code** (REQUIRED for services/buckets)
4. **Ensure consistency** - Variables match manifest inputs, outputs match manifest outputs
5. **Add validation** - Use Terraform validation blocks for input constraints

### 4. Testing Your Plugin

```bash
# Lint Terraform code
tflint --recursive

# Format Terraform code
terraform fmt -recursive

# Validate Terraform syntax
cd my-plugin/module
terraform init
terraform validate

# Test Go runtime code (for services/buckets)
cd ../..
go test ./my-plugin/runtime/go/...
go build ./my-plugin/runtime/go/...
```

### 5. Versioning and Releases

Plugin libraries use semantic versioning:

- **MAJOR** (v1.0.0 → v2.0.0): Breaking changes to plugin interfaces
- **MINOR** (v1.0.0 → v1.1.0): New plugins or new features to existing plugins
- **PATCH** (v1.0.0 → v1.0.1): Bug fixes, no interface changes

## Best Practices

### Plugin Design

1. **Keep plugins focused** - Each plugin should do one thing well
2. **Make inputs explicit** - Don't rely on implicit defaults from the provider
3. **Provide useful outputs** - Output values that downstream plugins or platforms need
4. **Use clear naming** - Plugin names should describe what they create
5. **Document thoroughly** - Descriptions in manifest and Terraform variables

### Terraform Implementation

1. **Use naming conventions** - Consistent resource naming across plugins
2. **Tag resources** - Add common tags for cost tracking and organization
3. **Handle errors gracefully** - Use validation blocks and preconditions
4. **Consider lifecycle** - Use lifecycle blocks when resources shouldn't be replaced
5. **Avoid hardcoding** - Use variables for all configurable values
6. **Export runtime config** - Use outputs to provide runtime configuration (e.g., bucket names, URLs)

### Go Runtime Code Guidelines

1. **Read from environment** - Use environment variables for configuration set by Terraform
2. **Handle credentials properly** - Use cloud provider credential chains
3. **Provide simple APIs** - Abstract away cloud-specific complexity
4. **Error handling** - Translate cloud errors to meaningful application errors
5. **Testing** - Include unit tests for runtime code
6. **Single go.mod** - Use the repository root go.mod for all runtime code

### Manifest Guidelines

1. **Required vs optional** - Mark inputs as required: true only when necessary
2. **Provide defaults** - Set sensible defaults for optional inputs
3. **Type safety** - Use correct types (string, number, array, object)
4. **Rich descriptions** - Help users understand what each input does
5. **Runtime declaration** - Services and buckets MUST declare runtime.go in manifest

## Example: Complete Service Plugin with Runtime Code

Here's a complete Lambda service plugin:

**`lambda/manifest.yaml`:**
```yaml
name: lambda
description: AWS Lambda serverless function
type: resource

required_identities:
  - aws

inputs:
  handler:
    type: string
    description: Lambda function handler
    required: true
    default: bootstrap

  timeout:
    type: number
    description: Function timeout in seconds
    required: false
    default: 30

  memory:
    type: number
    description: Memory in MB
    required: false
    default: 512

outputs:
  function_name:
    type: string
    description: Name of the Lambda function

  function_arn:
    type: string
    description: ARN of the Lambda function

  invoke_url:
    type: string
    description: URL to invoke the function

runtime:
  go:
    package: github.com/nitrictech/plugins-aws/lambda/runtime/go
```

**`lambda/module/main.tf`:**
```hcl
resource "aws_lambda_function" "function" {
  function_name = var.name
  role          = var.role_arn
  handler       = var.handler
  runtime       = "provided.al2"
  timeout       = var.timeout
  memory_size   = var.memory

  filename = var.deployment_package

  environment {
    variables = merge(
      var.environment_variables,
      {
        SUGA_FUNCTION_NAME = var.name
      }
    )
  }
}

resource "aws_lambda_function_url" "function_url" {
  function_name      = aws_lambda_function.function.function_name
  authorization_type = "NONE"
}
```

**`lambda/module/variables.tf`:**
```hcl
variable "name" {
  type        = string
  description = "Name of the Lambda function"
}

variable "handler" {
  type        = string
  description = "Lambda function handler"
  default     = "bootstrap"
}

variable "timeout" {
  type        = number
  description = "Function timeout in seconds"
  default     = 30
}

variable "memory" {
  type        = number
  description = "Memory in MB"
  default     = 512
}

variable "role_arn" {
  type        = string
  description = "IAM role ARN for the function"
}

variable "deployment_package" {
  type        = string
  description = "Path to deployment package"
}

variable "environment_variables" {
  type        = map(string)
  description = "Environment variables"
  default     = {}
}
```

**`lambda/module/outputs.tf`:**
```hcl
output "function_name" {
  value       = aws_lambda_function.function.function_name
  description = "Name of the Lambda function"
}

output "function_arn" {
  value       = aws_lambda_function.function.arn
  description = "ARN of the Lambda function"
}

output "invoke_url" {
  value       = aws_lambda_function_url.function_url.function_url
  description = "URL to invoke the function"
}
```

**`lambda/runtime/go/runtime.go`:**
```go
package lambdaruntime

import (
    "context"
    "fmt"
    "net/http"
    "os"

    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-lambda-go/events"
)

// Handler is the function signature for Lambda handlers
type Handler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

// Start begins the Lambda runtime with the provided handler
func Start(handler Handler) {
    lambda.Start(handler)
}

// Config reads runtime configuration from environment
type Config struct {
    FunctionName string
}

// LoadConfig loads the runtime configuration
func LoadConfig() (*Config, error) {
    functionName := os.Getenv("SUGA_FUNCTION_NAME")
    if functionName == "" {
        return nil, fmt.Errorf("SUGA_FUNCTION_NAME not set")
    }

    return &Config{
        FunctionName: functionName,
    }, nil
}

// WrapHTTPHandler wraps a standard http.Handler for Lambda
func WrapHTTPHandler(h http.Handler) Handler {
    return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
        // Convert API Gateway request to http.Request
        // Convert http.Response to API Gateway response
        // Implementation details...
        return events.APIGatewayProxyResponse{
            StatusCode: 200,
            Body:       "response body",
        }, nil
    }
}
```

**`go.mod` (at repository root):**
```go
module github.com/nitrictech/plugins-aws

go 1.21

require (
    github.com/aws/aws-lambda-go v1.41.0
    github.com/aws/aws-sdk-go-v2 v1.24.0
    github.com/aws/aws-sdk-go-v2/config v1.26.1
    github.com/aws/aws-sdk-go-v2/service/s3 v1.47.5
)
```

## Example: Complete Bucket Plugin with Runtime Code

Here's a complete S3 bucket plugin:

**`s3-bucket/manifest.yaml`:**
```yaml
name: s3-bucket
description: AWS S3 bucket for object storage
type: resource

inputs:
  bucket_name:
    type: string
    description: Name of the S3 bucket
    required: true

  versioning_enabled:
    type: boolean
    description: Enable versioning on the bucket
    required: false
    default: false

outputs:
  bucket_id:
    type: string
    description: The name of the bucket

  bucket_arn:
    type: string
    description: The ARN of the bucket

runtime:
  go:
    package: github.com/nitrictech/plugins-aws/s3-bucket/runtime/go
```

**`s3-bucket/runtime/go/bucket.go`:**
```go
package s3runtime

import (
    "bytes"
    "context"
    "fmt"
    "io"
    "os"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
    bucketName string
    s3Client   *s3.Client
}

// NewClient creates a new bucket client using environment configuration
func NewClient() (*Client, error) {
    bucketName := os.Getenv("SUGA_BUCKET_NAME")
    if bucketName == "" {
        return nil, fmt.Errorf("SUGA_BUCKET_NAME environment variable not set")
    }

    cfg, err := config.LoadDefaultConfig(context.Background())
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }

    return &Client{
        bucketName: bucketName,
        s3Client:   s3.NewFromConfig(cfg),
    }, nil
}

// Put uploads an object to the bucket
func (c *Client) Put(ctx context.Context, key string, data []byte) error {
    _, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: &c.bucketName,
        Key:    &key,
        Body:   bytes.NewReader(data),
    })
    return err
}

// Get retrieves an object from the bucket
func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
    result, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: &c.bucketName,
        Key:    &key,
    })
    if err != nil {
        return nil, err
    }
    defer result.Body.Close()

    return io.ReadAll(result.Body)
}

// Delete removes an object from the bucket
func (c *Client) Delete(ctx context.Context, key string) error {
    _, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: &c.bucketName,
        Key:    &key,
    })
    return err
}
```

## Publishing Plugin Libraries

Plugin libraries are typically published via:

1. **Git repositories** - Tagged releases (e.g., GitHub releases)
2. **Registry integration** - Uploaded to Suga platform for consumption

The Suga CLI and platform handle downloading and caching plugin libraries when they're referenced in `platform.yaml` files.

## Common Patterns

### Runtime Environment Variables

Terraform outputs should set environment variables for runtime code:

```hcl
# In main.tf for a service plugin
resource "aws_ecs_task_definition" "task" {
  # ... other config ...

  container_definitions = jsonencode([{
    environment = [
      {
        name  = "SUGA_BUCKET_NAME"
        value = var.bucket_name
      },
      {
        name  = "SUGA_SERVICE_URL"
        value = aws_lb.service.dns_name
      }
    ]
  }])
}
```

### VPC Dependencies

Many plugins need VPC resources:

```yaml
inputs:
  vpc_id:
    type: string
    description: VPC ID to deploy into
    required: true

  subnet_ids:
    type: array
    description: Subnet IDs for the resource
    required: true
```

### IAM Role Integration

Resource plugins often need IAM roles:

```yaml
required_identities:
  - aws

inputs:
  role_arn:
    type: string
    description: IAM role ARN for the resource
    required: true
```

## Troubleshooting

### Common Issues

1. **Missing runtime code** - Service or bucket plugin without runtime/go directory
   - Solution: Services and buckets MUST have Go runtime code

2. **Manifest/Terraform mismatch** - Variables don't match manifest inputs
   - Solution: Ensure every manifest input has a corresponding Terraform variable

3. **Missing runtime declaration** - Service/bucket manifest missing runtime.go section
   - Solution: Add runtime.go.package to manifest

4. **Runtime can't find configuration** - Application can't connect to infrastructure
   - Solution: Ensure Terraform sets appropriate environment variables (SUGA_*)

5. **Go module issues** - Runtime code won't build
   - Solution: Ensure go.mod at repository root includes all necessary dependencies

## Next Steps

After creating plugins:

1. **Test individually** - Ensure each plugin works in isolation
2. **Test Go runtime code** - Build and test the runtime code
3. **Test runtime integration** - Verify runtime can communicate with deployed infrastructure
4. **Create a platform** - Compose your plugins into a platform.yaml
5. **Test the platform** - Use the platform in a suga.yaml application
6. **Publish** - Tag a release and make available to your team

## Critical Reminders

- **Services MUST have Go runtime code** in `runtime/go/`
- **Buckets MUST have Go runtime code** in `runtime/go/`
- **Runtime code is currently ONLY available in Go** - no other languages supported yet
- **Infrastructure and identity plugins do NOT need runtime code**
- **go.mod file is at the repository root** - not in individual plugin directories

## Resources

- **Terraform Documentation** - https://www.terraform.io/docs
- **Provider Documentation** - Refer to specific provider docs (AWS, GCP, etc.)
- **Go Documentation** - https://go.dev/doc/
- **Suga Platform Development Guide** - See `suga://guides/platform-development` for how platforms consume plugins
