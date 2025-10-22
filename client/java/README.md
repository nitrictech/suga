# Suga Kotlin/Java Client

The official Kotlin/Java client library for [Suga](https://github.com/nitrictech/suga).

## Installation

Add the following dependency to your `pom.xml`:

```xml
<dependency>
    <groupId>com.addsuga.client</groupId>
    <artifactId>suga-client</artifactId>
    <version>0.0.1</version>
</dependency>
```

For Gradle, add to your `build.gradle`:

```gradle
implementation 'com.addsuga.client:suga-client:0.0.1'
```

## Usage

### Basic Example (Kotlin)

```kotlin
import com.addsuga.client.SugaClient
import com.addsuga.client.Bucket

fun main() {
    // Create a client
    val client = SugaClient()
    
    // Create a bucket instance using the public API
    val myBucket = client.createBucket("my-bucket")
    
    // Write data to the bucket
    val data = "Hello, World!".toByteArray()
    myBucket.write("example.txt", data)
    
    // Read data from the bucket
    val readData = myBucket.read("example.txt")
    println(String(readData))
    
    // Check if a file exists
    val exists = myBucket.exists("example.txt")
    println("File exists: $exists")
    
    // List files with a prefix
    val files = myBucket.list("example")
    println("Files: $files")
    
    // Delete a file
    myBucket.delete("example.txt")
    
    // Close the client (or use use() for automatic resource management)
    client.close()
}
```

### Basic Example (Java)

```java
import com.addsuga.client.SugaClient;
import com.addsuga.client.Bucket;
import java.util.List;

public class Example {
    public static void main(String[] args) {
        // Create a client
        SugaClient client = new SugaClient();
        
        // Create a bucket instance using the public API
        Bucket myBucket = client.createBucket("my-bucket");
        
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


### Presigned URLs (Kotlin)

Generate presigned URLs for secure file access:

```kotlin
import com.addsuga.client.Bucket
import java.time.Duration

// Get a download URL (valid for 5 minutes by default)
val downloadUrl = bucket.getDownloadURL("my-file.txt")

// Get an upload URL with custom expiry
val uploadUrl = bucket.getUploadURL("new-file.txt",
    Bucket.PresignUrlOptions.write(Duration.ofHours(1)))
```

### Presigned URLs (Java)

```java
import com.addsuga.client.Bucket;
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

**Kotlin:**
```kotlin
val client = SugaClient("your-server:50051")
```

**Java:**
```java
SugaClient client = new SugaClient("your-server:50051");
```


## Development

### Building from Source

This project now uses Kotlin with Java compatibility. Both languages are compiled together:

```bash
mvn clean compile
```

### Running Tests

```bash
mvn test
```