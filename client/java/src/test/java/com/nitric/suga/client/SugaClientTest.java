package com.nitric.suga.client;

import org.junit.Test;
import static org.junit.Assert.*;

/**
 * Unit tests for SugaClient
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

    @Test
    public void testCreateBucket() {
        SugaClient client = new SugaClient();
        
        // Test creating a bucket
        Bucket bucket = client.createBucket("test-bucket");
        assertNotNull("Bucket should not be null", bucket);
        assertEquals("Bucket name should match", "test-bucket", bucket.getName());
        
        client.close();
    }
}