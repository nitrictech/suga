package com.addsuga.client;

import com.nitric.suga.storage.v2.StorageGrpc;
import com.nitric.suga.storage.v2.StorageProto.*;
import com.google.protobuf.ByteString;
import com.google.protobuf.Duration;

import java.time.temporal.ChronoUnit;
import java.util.List;
import java.util.concurrent.TimeUnit;
import java.util.stream.Collectors;

/**
 * Bucket provides methods for interacting with cloud storage buckets
 */
public class Bucket {
    private final String name;
    private final StorageGrpc.StorageBlockingStub storageClient;

    public Bucket(StorageGrpc.StorageBlockingStub storageClient, String bucketName) {
        if (storageClient == null) {
            throw new IllegalArgumentException("Storage client cannot be null");
        }
        if (bucketName == null || bucketName.trim().isEmpty()) {
            throw new IllegalArgumentException("Bucket name cannot be null or empty");
        }
        this.storageClient = storageClient;
        this.name = bucketName;
    }

    /**
     * Read a file from the bucket
     * @param key The key of the file to read
     * @return The file contents as byte array
     * @throws IllegalArgumentException if key is null or empty
     * @throws RuntimeException if the operation fails
     */
    public byte[] read(String key) {
        if (key == null || key.trim().isEmpty()) {
            throw new IllegalArgumentException("Key cannot be null or empty");
        }
        
        StorageReadRequest request = StorageReadRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .build();

        try {
            StorageReadResponse response = storageClient.read(request);
            return response.getBody().toByteArray();
        } catch (io.grpc.StatusRuntimeException e) {
            throw new RuntimeException("Failed to read file '" + key + "' from bucket '" + name + "': " + e.getStatus().getDescription(), e);
        } catch (Exception e) {
            throw new RuntimeException("Failed to read file '" + key + "' from bucket '" + name + "'", e);
        }
    }

    /**
     * Write a file to the bucket
     * @param key The key of the file to write
     * @param data The file contents
     * @throws IllegalArgumentException if key is null or empty, or data is null
     * @throws RuntimeException if the operation fails
     */
    public void write(String key, byte[] data) {
        if (key == null || key.trim().isEmpty()) {
            throw new IllegalArgumentException("Key cannot be null or empty");
        }
        if (data == null) {
            throw new IllegalArgumentException("Data cannot be null");
        }
        
        StorageWriteRequest request = StorageWriteRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .setBody(ByteString.copyFrom(data))
                .build();

        try {
            storageClient.write(request);
        } catch (io.grpc.StatusRuntimeException e) {
            throw new RuntimeException("Failed to write file '" + key + "' to bucket '" + name + "': " + e.getStatus().getDescription(), e);
        } catch (Exception e) {
            throw new RuntimeException("Failed to write file '" + key + "' to bucket '" + name + "'", e);
        }
    }

    /**
     * Delete a file from the bucket
     * @param key The key of the file to delete
     * @throws IllegalArgumentException if key is null or empty
     * @throws RuntimeException if the operation fails
     */
    public void delete(String key) {
        if (key == null || key.trim().isEmpty()) {
            throw new IllegalArgumentException("Key cannot be null or empty");
        }
        
        StorageDeleteRequest request = StorageDeleteRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .build();

        try {
            storageClient.delete(request);
        } catch (io.grpc.StatusRuntimeException e) {
            throw new RuntimeException("Failed to delete file '" + key + "' from bucket '" + name + "': " + e.getStatus().getDescription(), e);
        } catch (Exception e) {
            throw new RuntimeException("Failed to delete file '" + key + "' from bucket '" + name + "'", e);
        }
    }

    /**
     * List files in the bucket with a given prefix
     * @param prefix The prefix to filter files (can be null or empty for all files)
     * @return List of file keys
     * @throws RuntimeException if the operation fails
     */
    public List<String> list(String prefix) {
        // Prefix can be null or empty, which means list all files
        String safePrefix = prefix == null ? "" : prefix;
        
        StorageListBlobsRequest request = StorageListBlobsRequest.newBuilder()
                .setBucketName(name)
                .setPrefix(safePrefix)
                .build();

        try {
            StorageListBlobsResponse response = storageClient.listBlobs(request);
            return response.getBlobsList().stream()
                    .map(StorageListBlobsResponse.Blob::getKey)
                    .collect(Collectors.toList());
        } catch (io.grpc.StatusRuntimeException e) {
            throw new RuntimeException("Failed to list files in bucket '" + name + "' with prefix '" + safePrefix + "': " + e.getStatus().getDescription(), e);
        } catch (Exception e) {
            throw new RuntimeException("Failed to list files in bucket '" + name + "' with prefix '" + safePrefix + "'", e);
        }
    }

    /**
     * Check if a file exists in the bucket
     * @param key The key of the file to check
     * @return true if the file exists, false otherwise
     * @throws IllegalArgumentException if key is null or empty
     * @throws RuntimeException if the operation fails
     */
    public boolean exists(String key) {
        if (key == null || key.trim().isEmpty()) {
            throw new IllegalArgumentException("Key cannot be null or empty");
        }
        
        StorageExistsRequest request = StorageExistsRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .build();

        try {
            StorageExistsResponse response = storageClient.exists(request);
            return response.getExists();
        } catch (io.grpc.StatusRuntimeException e) {
            throw new RuntimeException("Failed to check if file '" + key + "' exists in bucket '" + name + "': " + e.getStatus().getDescription(), e);
        } catch (Exception e) {
            throw new RuntimeException("Failed to check if file '" + key + "' exists in bucket '" + name + "'", e);
        }
    }

    /**
     * Mode for presigned URL operations
     */
    public enum Mode {
        READ,
        WRITE
    }

    /**
     * Options for presigned URL generation
     */
    public static class PresignUrlOptions {
        private final Mode mode;
        private final java.time.Duration expiry;

        public PresignUrlOptions(Mode mode, java.time.Duration expiry) {
            if (mode == null) {
                throw new IllegalArgumentException("Mode cannot be null");
            }
            if (expiry == null) {
                throw new IllegalArgumentException("Expiry cannot be null");
            }
            if (expiry.isNegative() || expiry.isZero()) {
                throw new IllegalArgumentException("Expiry must be positive");
            }
            this.mode = mode;
            this.expiry = expiry;
        }

        public Mode getMode() {
            return mode;
        }

        public java.time.Duration getExpiry() {
            return expiry;
        }

        public static PresignUrlOptions defaultRead() {
            return new PresignUrlOptions(Mode.READ, java.time.Duration.of(5, ChronoUnit.MINUTES));
        }

        public static PresignUrlOptions defaultWrite() {
            return new PresignUrlOptions(Mode.WRITE, java.time.Duration.of(5, ChronoUnit.MINUTES));
        }

        public static PresignUrlOptions read(java.time.Duration expiry) {
            return new PresignUrlOptions(Mode.READ, expiry);
        }

        public static PresignUrlOptions write(java.time.Duration expiry) {
            return new PresignUrlOptions(Mode.WRITE, expiry);
        }
    }

    private String preSignUrl(String key, PresignUrlOptions options) {
        if (key == null || key.trim().isEmpty()) {
            throw new IllegalArgumentException("Key cannot be null or empty");
        }
        if (options == null) {
            throw new IllegalArgumentException("Options cannot be null");
        }
        
        StoragePreSignUrlRequest.Operation operation = options.getMode() == Mode.WRITE
                ? StoragePreSignUrlRequest.Operation.WRITE
                : StoragePreSignUrlRequest.Operation.READ;

        Duration protobufDuration = Duration.newBuilder()
                .setSeconds(options.getExpiry().getSeconds())
                .setNanos((int) options.getExpiry().getNano())
                .build();

        StoragePreSignUrlRequest request = StoragePreSignUrlRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .setOperation(operation)
                .setExpiry(protobufDuration)
                .build();

        try {
            StoragePreSignUrlResponse response = storageClient.preSignUrl(request);
            return response.getUrl();
        } catch (io.grpc.StatusRuntimeException e) {
            throw new RuntimeException("Failed to get presigned URL for file '" + key + "' in bucket '" + name + "': " + e.getStatus().getDescription(), e);
        } catch (Exception e) {
            throw new RuntimeException("Failed to get presigned URL for file '" + key + "' in bucket '" + name + "'", e);
        }
    }

    /**
     * Get a presigned URL for downloading a file from the bucket
     * @param key The key of the file
     * @return The presigned download URL
     * @throws RuntimeException if the operation fails
     */
    public String getDownloadURL(String key) {
        return getDownloadURL(key, PresignUrlOptions.defaultRead());
    }

    /**
     * Get a presigned URL for downloading a file from the bucket with custom expiry
     * @param key The key of the file
     * @param options Options for URL generation
     * @return The presigned download URL
     * @throws IllegalArgumentException if key is null or empty, options is null, or mode is not READ
     * @throws RuntimeException if the operation fails
     */
    public String getDownloadURL(String key, PresignUrlOptions options) {
        if (options == null) {
            throw new IllegalArgumentException("Options cannot be null");
        }
        if (options.getMode() != Mode.READ) {
            throw new IllegalArgumentException("Options mode must be READ for download URLs, but was " + options.getMode());
        }
        return preSignUrl(key, options);
    }

    /**
     * Get a presigned URL for uploading a file to the bucket
     * @param key The key of the file
     * @return The presigned upload URL
     * @throws RuntimeException if the operation fails
     */
    public String getUploadURL(String key) {
        return getUploadURL(key, PresignUrlOptions.defaultWrite());
    }

    /**
     * Get a presigned URL for uploading a file to the bucket with custom expiry
     * @param key The key of the file
     * @param options Options for URL generation
     * @return The presigned upload URL
     * @throws IllegalArgumentException if key is null or empty, options is null, or mode is not WRITE
     * @throws RuntimeException if the operation fails
     */
    public String getUploadURL(String key, PresignUrlOptions options) {
        if (options == null) {
            throw new IllegalArgumentException("Options cannot be null");
        }
        if (options.getMode() != Mode.WRITE) {
            throw new IllegalArgumentException("Options mode must be WRITE for upload URLs, but was " + options.getMode());
        }
        return preSignUrl(key, options);
    }

    /**
     * Get the bucket name
     * @return The bucket name
     */
    public String getName() {
        return name;
    }
}