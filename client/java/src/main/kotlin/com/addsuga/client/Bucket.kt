package com.addsuga.client

import com.google.protobuf.ByteString
import com.google.protobuf.Duration
import io.suga.proto.storage.v2.*
import io.suga.proto.storage.v2.StorageGrpc
import java.time.temporal.ChronoUnit

/** Bucket provides methods for interacting with cloud storage buckets */
class Bucket(private val storageClient: StorageGrpc.StorageBlockingStub, private val name: String) {

  init {
    require(name.isNotBlank()) { "Bucket name cannot be null or empty" }
  }

  /**
   * Read a file from the bucket
   *
   * @param key The key of the file to read
   * @return The file contents as byte array
   * @throws IllegalArgumentException if key is null or empty
   * @throws RuntimeException if the operation fails
   */
  fun read(key: String): ByteArray {
    require(key.isNotBlank()) { "Key cannot be null or empty" }

    val request = StorageReadRequest.newBuilder().setBucketName(name).setKey(key).build()

    return try {
      val response = storageClient.read(request)
      response.body.toByteArray()
    } catch (e: io.grpc.StatusRuntimeException) {
      throw RuntimeException(
        "Failed to read file '$key' from bucket '$name': ${e.status.description}",
        e
      )
    } catch (e: Exception) {
      throw RuntimeException("Failed to read file '$key' from bucket '$name'", e)
    }
  }

  /**
   * Write a file to the bucket
   *
   * @param key The key of the file to write
   * @param data The file contents
   * @throws IllegalArgumentException if key is null or empty, or data is null
   * @throws RuntimeException if the operation fails
   */
  fun write(key: String, data: ByteArray) {
    require(key.isNotBlank()) { "Key cannot be null or empty" }

    val request =
      StorageWriteRequest.newBuilder()
        .setBucketName(name)
        .setKey(key)
        .setBody(ByteString.copyFrom(data))
        .build()

    try {
      storageClient.write(request)
    } catch (e: io.grpc.StatusRuntimeException) {
      throw RuntimeException(
        "Failed to write file '$key' to bucket '$name': ${e.status.description}",
        e
      )
    } catch (e: Exception) {
      throw RuntimeException("Failed to write file '$key' to bucket '$name'", e)
    }
  }

  /**
   * Delete a file from the bucket
   *
   * @param key The key of the file to delete
   * @throws IllegalArgumentException if key is null or empty
   * @throws RuntimeException if the operation fails
   */
  fun delete(key: String) {
    require(key.isNotBlank()) { "Key cannot be null or empty" }

    val request = StorageDeleteRequest.newBuilder().setBucketName(name).setKey(key).build()

    try {
      storageClient.delete(request)
    } catch (e: io.grpc.StatusRuntimeException) {
      throw RuntimeException(
        "Failed to delete file '$key' from bucket '$name': ${e.status.description}",
        e
      )
    } catch (e: Exception) {
      throw RuntimeException("Failed to delete file '$key' from bucket '$name'", e)
    }
  }

  /**
   * List files in the bucket with a given prefix
   *
   * @param prefix The prefix to filter files (can be null or empty for all files)
   * @return List of file keys
   * @throws RuntimeException if the operation fails
   */
  fun list(prefix: String? = null): List<String> {
    val safePrefix = prefix ?: ""

    val request =
      StorageListBlobsRequest.newBuilder().setBucketName(name).setPrefix(safePrefix).build()

    return try {
      val response = storageClient.listBlobs(request)
      response.blobsList.map { it.key }
    } catch (e: io.grpc.StatusRuntimeException) {
      throw RuntimeException(
        "Failed to list files in bucket '$name' with prefix '$safePrefix': ${e.status.description}",
        e
      )
    } catch (e: Exception) {
      throw RuntimeException("Failed to list files in bucket '$name' with prefix '$safePrefix'", e)
    }
  }

  /**
   * Check if a file exists in the bucket
   *
   * @param key The key of the file to check
   * @return true if the file exists, false otherwise
   * @throws IllegalArgumentException if key is null or empty
   * @throws RuntimeException if the operation fails
   */
  fun exists(key: String): Boolean {
    require(key.isNotBlank()) { "Key cannot be null or empty" }

    val request = StorageExistsRequest.newBuilder().setBucketName(name).setKey(key).build()

    return try {
      val response = storageClient.exists(request)
      response.exists
    } catch (e: io.grpc.StatusRuntimeException) {
      throw RuntimeException(
        "Failed to check if file '$key' exists in bucket '$name': ${e.status.description}",
        e
      )
    } catch (e: Exception) {
      throw RuntimeException("Failed to check if file '$key' exists in bucket '$name'", e)
    }
  }

  /** Mode for presigned URL operations */
  enum class Mode {
    READ,
    WRITE
  }

  /** Options for presigned URL generation */
  data class PresignUrlOptions(val mode: Mode, val expiry: java.time.Duration) {
    init {
      require(!expiry.isNegative && !expiry.isZero) { "Expiry must be positive" }
    }

    companion object {
      @JvmStatic
      fun defaultRead(): PresignUrlOptions =
        PresignUrlOptions(Mode.READ, java.time.Duration.of(5, ChronoUnit.MINUTES))

      @JvmStatic
      fun defaultWrite(): PresignUrlOptions =
        PresignUrlOptions(Mode.WRITE, java.time.Duration.of(5, ChronoUnit.MINUTES))

      @JvmStatic
      fun read(expiry: java.time.Duration): PresignUrlOptions = PresignUrlOptions(Mode.READ, expiry)

      @JvmStatic
      fun write(expiry: java.time.Duration): PresignUrlOptions =
        PresignUrlOptions(Mode.WRITE, expiry)
    }
  }

  private fun preSignUrl(key: String, options: PresignUrlOptions): String {
    require(key.isNotBlank()) { "Key cannot be null or empty" }

    val operation =
      if (options.mode == Mode.WRITE) {
        StoragePreSignUrlRequest.Operation.WRITE
      } else {
        StoragePreSignUrlRequest.Operation.READ
      }

    val protobufDuration =
      Duration.newBuilder().setSeconds(options.expiry.seconds).setNanos(options.expiry.nano).build()

    val request =
      StoragePreSignUrlRequest.newBuilder()
        .setBucketName(name)
        .setKey(key)
        .setOperation(operation)
        .setExpiry(protobufDuration)
        .build()

    return try {
      val response = storageClient.preSignUrl(request)
      response.url
    } catch (e: io.grpc.StatusRuntimeException) {
      throw RuntimeException(
        "Failed to get presigned URL for file '$key' in bucket '$name': ${e.status.description}",
        e
      )
    } catch (e: Exception) {
      throw RuntimeException("Failed to get presigned URL for file '$key' in bucket '$name'", e)
    }
  }

  /**
   * Get a presigned URL for downloading a file from the bucket
   *
   * @param key The key of the file
   * @return The presigned download URL
   * @throws RuntimeException if the operation fails
   */
  fun getDownloadURL(key: String): String = getDownloadURL(key, PresignUrlOptions.defaultRead())

  /**
   * Get a presigned URL for downloading a file from the bucket with custom expiry
   *
   * @param key The key of the file
   * @param options Options for URL generation
   * @return The presigned download URL
   * @throws IllegalArgumentException if key is null or empty, or mode is not READ
   * @throws RuntimeException if the operation fails
   */
  fun getDownloadURL(key: String, options: PresignUrlOptions): String {
    require(options.mode == Mode.READ) {
      "Options mode must be READ for download URLs, but was ${options.mode}"
    }
    return preSignUrl(key, options)
  }

  /**
   * Get a presigned URL for uploading a file to the bucket
   *
   * @param key The key of the file
   * @return The presigned upload URL
   * @throws RuntimeException if the operation fails
   */
  fun getUploadURL(key: String): String = getUploadURL(key, PresignUrlOptions.defaultWrite())

  /**
   * Get a presigned URL for uploading a file to the bucket with custom expiry
   *
   * @param key The key of the file
   * @param options Options for URL generation
   * @return The presigned upload URL
   * @throws IllegalArgumentException if key is null or empty, or mode is not write
   * @throws RuntimeException if the operation fails
   */
  fun getUploadURL(key: String, options: PresignUrlOptions): String {
    require(options.mode == Mode.WRITE) {
      "Options mode must be WRITE for upload URLs, but was ${options.mode}"
    }
    return preSignUrl(key, options)
  }

  /**
   * Get the bucket name
   *
   * @return The bucket name
   */
  fun getName(): String = name
}
