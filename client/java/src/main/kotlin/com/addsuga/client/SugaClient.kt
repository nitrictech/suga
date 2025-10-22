package com.addsuga.client

import io.grpc.ManagedChannel
import io.grpc.ManagedChannelBuilder
import io.suga.proto.storage.v2.StorageGrpc
import java.util.concurrent.TimeUnit

/**
 * Base SugaClient class that provides connection management for Suga runtime services. This class
 * should be extended by generated client classes.
 */
open class SugaClient : AutoCloseable {
  private val channel: ManagedChannel
  private val storageClient: StorageGrpc.StorageBlockingStub

  /** Create a new SugaClient with default configuration */
  constructor() : this(getServiceAddress())

  /**
   * Create a new SugaClient with custom server address
   *
   * @param address The gRPC server address (e.g., "localhost:50051")
   * @throws IllegalArgumentException if address is null or empty
   */
  constructor(address: String) {
    require(address.isNotBlank()) { "Address cannot be null or empty" }

    val isSecure = System.getenv("SUGA_SERVICE_SECURE")?.let {
      it.trim().equals("true", ignoreCase = true)
    } ?: false

    val channelBuilder = ManagedChannelBuilder.forTarget(address)
    this.channel = if (isSecure) {
      channelBuilder.useTransportSecurity().build()
    } else {
      channelBuilder.usePlaintext().build()
    }
    
    this.storageClient = StorageGrpc.newBlockingStub(channel)
      .withDeadlineAfter(10, TimeUnit.SECONDS)
  }

  /**
   * Get the storage client for bucket operations
   *
   * @return StorageGrpc.StorageBlockingStub instance
   */
  protected fun getStorageClient(): StorageGrpc.StorageBlockingStub = storageClient

  /**
   * Create a new Bucket instance
   *
   * @param bucketName The name of the bucket
   * @return A new Bucket instance
   * @throws IllegalArgumentException if bucketName is null or empty
   */
  fun createBucket(bucketName: String): Bucket {
    require(bucketName.isNotBlank()) { "Bucket name cannot be null or empty" }
    return Bucket(storageClient, bucketName)
  }

  /**
   * Close the client and release resources
   *
   * @throws RuntimeException if shutdown fails or times out
   */
  override fun close() {
    if (!channel.isShutdown) {
      channel.shutdown()
      try {
        // Wait for the channel to terminate within a reasonable timeout
        if (!channel.awaitTermination(5, TimeUnit.SECONDS)) {
          // Force shutdown if graceful shutdown didn't complete in time
          channel.shutdownNow()
          if (!channel.awaitTermination(5, TimeUnit.SECONDS)) {
            throw RuntimeException("Failed to shutdown gRPC channel within timeout")
          }
        }
      } catch (e: InterruptedException) {
        // Restore interrupted status
        Thread.currentThread().interrupt()
        // Force shutdown
        channel.shutdownNow()
        throw RuntimeException("Interrupted while shutting down gRPC channel", e)
      }
    }
  }

  companion object {
    /**
     * Get the service address from environment variable or use default
     *
     * @return The service address
     */
    private fun getServiceAddress(): String {
      val address = System.getenv("SUGA_SERVICE_ADDRESS")
      return if (address.isNullOrBlank()) "localhost:50051" else address
    }
  }
}
