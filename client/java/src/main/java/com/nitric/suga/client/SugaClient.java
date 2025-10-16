package com.nitric.suga.client;

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
     */
    public SugaClient(String address) {
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
     */
    protected Bucket createBucket(String bucketName) {
        return new Bucket(storageClient, bucketName);
    }

    /**
     * Close the client and release resources
     */
    public void close() {
        if (channel != null) {
            channel.shutdown();
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