package com.nitric.suga.client;

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
        this.storageClient = storageClient;
        this.name = bucketName;
    }

    /**
     * Read a file from the bucket
     * @param key The key of the file to read
     * @return The file contents as byte array
     * @throws RuntimeException if the operation fails
     */
    public byte[] read(String key) {
        StorageReadRequest request = StorageReadRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .build();

        try {
            StorageReadResponse response = storageClient.read(request);
            return response.getBody().toByteArray();
        } catch (Exception e) {
            throw new RuntimeException("Failed to read file from the " + name + " bucket", e);
        }
    }

    /**
     * Write a file to the bucket
     * @param key The key of the file to write
     * @param data The file contents
     * @throws RuntimeException if the operation fails
     */
    public void write(String key, byte[] data) {
        StorageWriteRequest request = StorageWriteRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .setBody(ByteString.copyFrom(data))
                .build();

        try {
            storageClient.write(request);
        } catch (Exception e) {
            throw new RuntimeException("Failed to write file to bucket", e);
        }
    }

    /**
     * Delete a file from the bucket
     * @param key The key of the file to delete
     * @throws RuntimeException if the operation fails
     */
    public void delete(String key) {
        StorageDeleteRequest request = StorageDeleteRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .build();

        try {
            storageClient.delete(request);
        } catch (Exception e) {
            throw new RuntimeException("Failed to delete file from bucket", e);
        }
    }

    /**
     * List files in the bucket with a given prefix
     * @param prefix The prefix to filter files
     * @return List of file keys
     * @throws RuntimeException if the operation fails
     */
    public List<String> list(String prefix) {
        StorageListBlobsRequest request = StorageListBlobsRequest.newBuilder()
                .setBucketName(name)
                .setPrefix(prefix)
                .build();

        try {
            StorageListBlobsResponse response = storageClient.listBlobs(request);
            return response.getBlobsList().stream()
                    .map(StorageListBlobsResponse.Blob::getKey)
                    .collect(Collectors.toList());
        } catch (Exception e) {
            throw new RuntimeException("Failed to list files in bucket", e);
        }
    }

    /**
     * Check if a file exists in the bucket
     * @param key The key of the file to check
     * @return true if the file exists, false otherwise
     * @throws RuntimeException if the operation fails
     */
    public boolean exists(String key) {
        StorageExistsRequest request = StorageExistsRequest.newBuilder()
                .setBucketName(name)
                .setKey(key)
                .build();

        try {
            StorageExistsResponse response = storageClient.exists(request);
            return response.getExists();
        } catch (Exception e) {
            throw new RuntimeException("Failed to check if file exists in bucket", e);
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
        } catch (Exception e) {
            throw new RuntimeException("Failed to get presigned URL for file", e);
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
     * @throws RuntimeException if the operation fails
     */
    public String getDownloadURL(String key, PresignUrlOptions options) {
        PresignUrlOptions readOptions = new PresignUrlOptions(Mode.READ, options.getExpiry());
        return preSignUrl(key, readOptions);
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
     * @throws RuntimeException if the operation fails
     */
    public String getUploadURL(String key, PresignUrlOptions options) {
        PresignUrlOptions writeOptions = new PresignUrlOptions(Mode.WRITE, options.getExpiry());
        return preSignUrl(key, writeOptions);
    }

    /**
     * Get the bucket name
     * @return The bucket name
     */
    public String getName() {
        return name;
    }
}