package com.addsuga.client;

import com.nitric.suga.storage.v2.StorageGrpc;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;

/**
 * Base SugaClient class that provides connection management for Suga runtime services.
 * This class should be extended by generated client classes.
 */
public class SugaClient {
    private final ManagedChannel channel;
    private final StorageGrpc.StorageBlockingStub storageClient;

    /**
     * Create a new SugaClient with default configuration
     */
    public SugaClient() {
        this(getServiceAddress());
    }

    /**
     * Create a new SugaClient with custom server address
     * @param address The gRPC server address (e.g., "localhost:50051")
     * @throws IllegalArgumentException if address is null or empty
     */
    public SugaClient(String address) {
        if (address == null || address.trim().isEmpty()) {
            throw new IllegalArgumentException("Address cannot be null or empty");
        }
        this.channel = ManagedChannelBuilder.forTarget(address)
                .usePlaintext()
                .build();
        this.storageClient = StorageGrpc.newBlockingStub(channel);
    }

    /**
     * Get the storage client for bucket operations
     * @return StorageGrpc.StorageBlockingStub instance
     */
    protected StorageGrpc.StorageBlockingStub getStorageClient() {
        return storageClient;
    }

    /**
     * Create a new Bucket instance
     * @param bucketName The name of the bucket
     * @return A new Bucket instance
     * @throws IllegalArgumentException if bucketName is null or empty
     */
    public Bucket createBucket(String bucketName) {
        if (bucketName == null || bucketName.trim().isEmpty()) {
            throw new IllegalArgumentException("Bucket name cannot be null or empty");
        }
        return new Bucket(storageClient, bucketName);
    }

    /**
     * Close the client and release resources
     * @throws RuntimeException if shutdown fails or times out
     */
    public void close() {
        if (channel != null && !channel.isShutdown()) {
            channel.shutdown();
            try {
                // Wait for the channel to terminate within a reasonable timeout
                if (!channel.awaitTermination(5, java.util.concurrent.TimeUnit.SECONDS)) {
                    // Force shutdown if graceful shutdown didn't complete in time
                    channel.shutdownNow();
                    if (!channel.awaitTermination(5, java.util.concurrent.TimeUnit.SECONDS)) {
                        throw new RuntimeException("Failed to shutdown gRPC channel within timeout");
                    }
                }
            } catch (InterruptedException e) {
                // Restore interrupted status
                Thread.currentThread().interrupt();
                // Force shutdown
                channel.shutdownNow();
                throw new RuntimeException("Interrupted while shutting down gRPC channel", e);
            }
        }
    }

    /**
     * Get the service address from environment variable or use default
     * @return The service address
     */
    private static String getServiceAddress() {
        String address = System.getenv("SUGA_SERVICE_ADDRESS");
        if (address == null || address.trim().isEmpty()) {
            address = "localhost:50051";
        }
        return address;
    }
}