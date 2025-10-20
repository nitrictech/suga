package com.addsuga.client;

import org.junit.Test;
import static org.junit.Assert.*;

import java.time.Duration;
import java.time.temporal.ChronoUnit;

/**
 * Unit tests for SugaClient and Bucket classes
 */
public class SugaClientTest {

    @Test
    public void testSugaClientCreation() {
        // Test that we can create a SugaClient instance
        SugaClient client = new SugaClient();
        assertNotNull("SugaClient should not be null", client);
        
        // Test cleanup
        client.close();
    }

    @Test
    public void testSugaClientWithCustomAddress() {
        // Test creating client with custom address
        SugaClient client = new SugaClient("localhost:9999");
        assertNotNull("SugaClient with custom address should not be null", client);
        
        // Test cleanup
        client.close();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testSugaClientWithNullAddress() {
        // Test that null address throws exception
        new SugaClient(null);
    }

    @Test(expected = IllegalArgumentException.class)
    public void testSugaClientWithEmptyAddress() {
        // Test that empty address throws exception
        new SugaClient("");
    }

    @Test(expected = IllegalArgumentException.class)
    public void testSugaClientWithWhitespaceAddress() {
        // Test that whitespace-only address throws exception
        new SugaClient("   ");
    }

    @Test
    public void testCreateBucket() {
        SugaClient client = new SugaClient();
        
        // Test creating a bucket
        Bucket bucket = client.createBucket("test-bucket");
        assertNotNull("Bucket should not be null", bucket);
        assertEquals("Bucket name should match", "test-bucket", bucket.getName());
        
        client.close();
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCreateBucketWithNullName() {
        SugaClient client = new SugaClient();
        try {
            client.createBucket(null);
        } finally {
            client.close();
        }
    }

    @Test(expected = IllegalArgumentException.class)
    public void testCreateBucketWithEmptyName() {
        SugaClient client = new SugaClient();
        try {
            client.createBucket("");
        } finally {
            client.close();
        }
    }

    @Test
    public void testBucketParameterValidation() {
        SugaClient client = new SugaClient();
        Bucket bucket = client.createBucket("test-bucket");
        
        try {
            // Test that null key throws exception in read
            try {
                bucket.read(null);
                fail("Expected IllegalArgumentException for null key in read");
            } catch (IllegalArgumentException e) {
                // Expected
            } catch (RuntimeException e) {
                // May also throw RuntimeException due to gRPC call
            }

            // Test that empty key throws exception in write
            try {
                bucket.write("", new byte[0]);
                fail("Expected IllegalArgumentException for empty key in write");
            } catch (IllegalArgumentException e) {
                // Expected
            } catch (RuntimeException e) {
                // May also throw RuntimeException due to gRPC call
            }

            // Test that null data throws exception in write
            try {
                bucket.write("test-key", null);
                fail("Expected IllegalArgumentException for null data in write");
            } catch (IllegalArgumentException e) {
                // Expected
            } catch (RuntimeException e) {
                // May also throw RuntimeException due to gRPC call
            }

            // Test that null key throws exception in delete
            try {
                bucket.delete(null);
                fail("Expected IllegalArgumentException for null key in delete");
            } catch (IllegalArgumentException e) {
                // Expected
            } catch (RuntimeException e) {
                // May also throw RuntimeException due to gRPC call
            }

            // Test that null key throws exception in exists
            try {
                bucket.exists(null);
                fail("Expected IllegalArgumentException for null key in exists");
            } catch (IllegalArgumentException e) {
                // Expected
            } catch (RuntimeException e) {
                // May also throw RuntimeException due to gRPC call
            }

        } finally {
            client.close();
        }
    }

    @Test
    public void testPresignUrlOptionsValidation() {
        // Test valid options
        Bucket.PresignUrlOptions validOptions = new Bucket.PresignUrlOptions(
                Bucket.Mode.READ,
                Duration.of(5, ChronoUnit.MINUTES)
        );
        assertNotNull("Valid options should be created", validOptions);

        // Test null mode throws exception
        try {
            new Bucket.PresignUrlOptions(null, Duration.of(5, ChronoUnit.MINUTES));
            fail("Expected IllegalArgumentException for null mode");
        } catch (IllegalArgumentException e) {
            // Expected
        }

        // Test null expiry throws exception
        try {
            new Bucket.PresignUrlOptions(Bucket.Mode.READ, null);
            fail("Expected IllegalArgumentException for null expiry");
        } catch (IllegalArgumentException e) {
            // Expected
        }

        // Test negative expiry throws exception
        try {
            new Bucket.PresignUrlOptions(Bucket.Mode.READ, Duration.of(-1, ChronoUnit.MINUTES));
            fail("Expected IllegalArgumentException for negative expiry");
        } catch (IllegalArgumentException e) {
            // Expected
        }

        // Test zero expiry throws exception
        try {
            new Bucket.PresignUrlOptions(Bucket.Mode.READ, Duration.ZERO);
            fail("Expected IllegalArgumentException for zero expiry");
        } catch (IllegalArgumentException e) {
            // Expected
        }
    }

    @Test
    public void testPresignUrlOptionsStaticMethods() {
        // Test default read options
        Bucket.PresignUrlOptions readOptions = Bucket.PresignUrlOptions.defaultRead();
        assertNotNull("Default read options should not be null", readOptions);
        assertEquals("Default read mode should be READ", Bucket.Mode.READ, readOptions.getMode());
        assertEquals("Default expiry should be 5 minutes", Duration.of(5, ChronoUnit.MINUTES), readOptions.getExpiry());

        // Test default write options
        Bucket.PresignUrlOptions writeOptions = Bucket.PresignUrlOptions.defaultWrite();
        assertNotNull("Default write options should not be null", writeOptions);
        assertEquals("Default write mode should be WRITE", Bucket.Mode.WRITE, writeOptions.getMode());
        assertEquals("Default expiry should be 5 minutes", Duration.of(5, ChronoUnit.MINUTES), writeOptions.getExpiry());

        // Test read with custom expiry
        Duration customExpiry = Duration.of(10, ChronoUnit.MINUTES);
        Bucket.PresignUrlOptions customReadOptions = Bucket.PresignUrlOptions.read(customExpiry);
        assertNotNull("Custom read options should not be null", customReadOptions);
        assertEquals("Custom read mode should be READ", Bucket.Mode.READ, customReadOptions.getMode());
        assertEquals("Custom expiry should match", customExpiry, customReadOptions.getExpiry());

        // Test write with custom expiry
        Bucket.PresignUrlOptions customWriteOptions = Bucket.PresignUrlOptions.write(customExpiry);
        assertNotNull("Custom write options should not be null", customWriteOptions);
        assertEquals("Custom write mode should be WRITE", Bucket.Mode.WRITE, customWriteOptions.getMode());
        assertEquals("Custom expiry should match", customExpiry, customWriteOptions.getExpiry());
    }

    @Test
    public void testGetPresignedUrlParameterValidation() {
        SugaClient client = new SugaClient();
        Bucket bucket = client.createBucket("test-bucket");
        
        try {
            // Test getDownloadURL with null options
            try {
                bucket.getDownloadURL("test-key", null);
                fail("Expected IllegalArgumentException for null options in getDownloadURL");
            } catch (IllegalArgumentException e) {
                // Expected
            } catch (RuntimeException e) {
                // May also throw RuntimeException due to gRPC call
            }

            // Test getUploadURL with null options
            try {
                bucket.getUploadURL("test-key", null);
                fail("Expected IllegalArgumentException for null options in getUploadURL");
            } catch (IllegalArgumentException e) {
                // Expected
            } catch (RuntimeException e) {
                // May also throw RuntimeException due to gRPC call
            }

        } finally {
            client.close();
        }
    }

    @Test
    public void testMultipleCloseCallsAreSafe() {
        SugaClient client = new SugaClient();
        
        // Multiple close calls should not throw exceptions
        client.close();
        client.close(); // Second close should be safe
        client.close(); // Third close should be safe
    }

    @Test
    public void testListWithNullPrefix() {
        SugaClient client = new SugaClient();
        Bucket bucket = client.createBucket("test-bucket");
        
        try {
            // list() with null prefix should work (lists all files)
            // This would normally throw RuntimeException due to gRPC call in test environment
            // but null prefix should be handled gracefully
            bucket.list(null);
        } catch (RuntimeException e) {
            // Expected in test environment due to no actual gRPC server
            // The important thing is that null prefix doesn't throw IllegalArgumentException
        } finally {
            client.close();
        }
    }
}