# Suga Java Client

The official Java client library for [Suga](https://github.com/nitrictech/suga).

## Installation

Add the following dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>com.nitric</groupId>
    <artifactId>suga-client</artifactId>
    <version>0.0.1</version>
</dependency>
```

For Gradle, add to your `build.gradle`:

```gradle
implementation 'com.nitric:suga-client:0.0.1'
```

## Usage

### Basic Example

```java
import com.nitric.suga.client.SugaClient;
import com.nitric.suga.client.Bucket;

public class Example {
    public static void main(String[] args) {
        // Create a client
        SugaClient client = new SugaClient();
        
        // Create a bucket instance
        Bucket myBucket = new Bucket(client.getStorageClient(), "my-bucket");
        
        // Write data to the bucket
        byte[] data = "Hello, World!".getBytes();
        myBucket.write("example.txt", data);
        
        // Read data from the bucket
        byte[] readData = myBucket.read("example.txt");
        System.out.println(new String(readData));
        
        // Check if a file exists
        boolean exists = myBucket.exists("example.txt");
        System.out.println("File exists: " + exists);
        
        // List files with a prefix
        List<String> files = myBucket.list("example");
        System.out.println("Files: " + files);
        
        // Delete a file
        myBucket.delete("example.txt");
        
        // Close the client
        client.close();
    }
}
```

### Generated Client Example

When using the Suga CLI to generate client code, you'll get a generated client class:

```java
import com.example.GeneratedSugaClient;

public class GeneratedExample {
    public static void main(String[] args) {
        // Create the generated client
        GeneratedSugaClient client = new GeneratedSugaClient();
        
        // Access your buckets directly (assuming you have a bucket named "my-bucket")
        client.myBucket.write("test.txt", "Hello from generated client!".getBytes());
        
        byte[] data = client.myBucket.read("test.txt");
        System.out.println(new String(data));
        
        // Close the client
        client.close();
    }
}
```

### Presigned URLs

Generate presigned URLs for secure file access:

```java
import com.nitric.suga.client.Bucket;
import java.time.Duration;

// Get a download URL (valid for 5 minutes by default)
String downloadUrl = bucket.getDownloadURL("my-file.txt");

// Get an upload URL with custom expiry
String uploadUrl = bucket.getUploadURL("new-file.txt", 
    Bucket.PresignUrlOptions.write(Duration.ofHours(1)));
```

## Configuration

The client uses the `SUGA_SERVICE_ADDRESS` environment variable to connect to the Suga runtime server. If not set, it defaults to `localhost:50051`.

```bash
export SUGA_SERVICE_ADDRESS=your-server:50051
```

Or set it programmatically:

```java
SugaClient client = new SugaClient("your-server:50051");
```

## API Reference

### SugaClient

The base client class that manages the gRPC connection to the Suga runtime.

#### Methods

- `SugaClient()` - Create a client with default configuration
- `SugaClient(String address)` - Create a client with custom server address
- `void close()` - Close the client and release resources

### Bucket

Provides methods for interacting with cloud storage buckets.

#### Methods

- `byte[] read(String key)` - Read a file from the bucket
- `void write(String key, byte[] data)` - Write a file to the bucket
- `void delete(String key)` - Delete a file from the bucket
- `List<String> list(String prefix)` - List files with a given prefix
- `boolean exists(String key)` - Check if a file exists
- `String getDownloadURL(String key)` - Get a presigned download URL
- `String getUploadURL(String key)` - Get a presigned upload URL
- `String getDownloadURL(String key, PresignUrlOptions options)` - Get download URL with options
- `String getUploadURL(String key, PresignUrlOptions options)` - Get upload URL with options

## Development

### Building from Source

```bash
mvn clean compile
```

### Running Tests

```bash
mvn test
```

### Generating Protobuf Sources

The protobuf sources are automatically generated during the build process. To manually generate them:

```bash
mvn protobuf:compile protobuf:compile-custom
```

## License

This project is licensed under the MPL-2.0 License - see the [LICENSE](../../LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.