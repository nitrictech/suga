package com.addsuga.client

import org.junit.Test
import org.junit.Assert.*
import java.time.Duration
import java.time.temporal.ChronoUnit

/**
 * Test to verify Kotlin syntax and language features work correctly
 */
class KotlinSyntaxTest {

    @Test
    fun testKotlinBasicSyntax() {
        // Test Kotlin null safety
        val notNullString: String = "Hello Kotlin"
        assertNotNull("String should not be null", notNullString)
        assertEquals("Hello Kotlin", notNullString)
        
        // Test Kotlin string interpolation
        val name = "Suga"
        val message = "Welcome to $name!"
        assertEquals("Welcome to Suga!", message)
    }

    @Test
    fun testKotlinDataClass() {
        // Test data class creation and properties
        data class TestConfig(val name: String, val version: Int)
        
        val config = TestConfig("test", 1)
        assertEquals("test", config.name)
        assertEquals(1, config.version)
        
        // Test data class copy function
        val updatedConfig = config.copy(version = 2)
        assertEquals("test", updatedConfig.name)
        assertEquals(2, updatedConfig.version)
    }

    @Test
    fun testKotlinRequireValidation() {
        // Test Kotlin's require function
        fun validateInput(input: String): String {
            require(input.isNotBlank()) { "Input cannot be blank" }
            return input.uppercase()
        }
        
        // Valid input should work
        assertEquals("HELLO", validateInput("hello"))
        
        // Invalid input should throw exception
        try {
            validateInput("")
            fail("Should have thrown IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue("Exception message should contain 'blank'", 
                      e.message?.contains("blank") == true)
        }
    }

    @Test
    fun testKotlinCollections() {
        // Test Kotlin collection functions
        val numbers = listOf(1, 2, 3, 4, 5)
        
        val doubled = numbers.map { it * 2 }
        assertEquals(listOf(2, 4, 6, 8, 10), doubled)
        
        val filtered = numbers.filter { it > 3 }
        assertEquals(listOf(4, 5), filtered)
        
        val sum = numbers.reduce { acc, n -> acc + n }
        assertEquals(15, sum)
    }

    @Test
    fun testKotlinWhenExpression() {
        fun describe(x: Any): String = when (x) {
            is String -> "String of length ${x.length}"
            is Int -> if (x > 0) "Positive integer" else "Non-positive integer"
            is List<*> -> "List with ${x.size} elements"
            else -> "Unknown type"
        }
        
        assertEquals("String of length 5", describe("hello"))
        assertEquals("Positive integer", describe(42))
        assertEquals("Non-positive integer", describe(-1))
        assertEquals("List with 3 elements", describe(listOf(1, 2, 3)))
        assertEquals("Unknown type", describe(Duration.ZERO))
    }

    @Test
    fun testKotlinExtensionFunction() {
        // Test extension function
        fun String.isPalindrome(): Boolean {
            val cleaned = this.lowercase().replace(Regex("[^a-z]"), "")
            return cleaned == cleaned.reversed()
        }
        
        assertTrue("racecar".isPalindrome())
        assertTrue("A man a plan a canal Panama".isPalindrome())
        assertFalse("hello".isPalindrome())
    }

    // Test classes moved outside function scope to avoid local class restrictions
    class TestClass {
        companion object {
            @JvmStatic
            fun createDefault(): TestClass = TestClass()
            
            const val DEFAULT_NAME = "default"
        }
        
        val name: String = DEFAULT_NAME
    }

    // Sealed class hierarchy for testing
    sealed class Result<out T>
    data class Success<T>(val value: T) : Result<T>()
    data class Error(val message: String) : Result<Nothing>()

    @Test
    fun testKotlinCompanionObject() {
        val instance = TestClass.createDefault()
        assertEquals("default", instance.name)
        assertEquals("default", TestClass.DEFAULT_NAME)
    }

    @Test
    fun testKotlinSealedClass() {
        fun processResult(result: Result<String>): String = when (result) {
            is Success -> "Got: ${result.value}"
            is Error -> "Error: ${result.message}"
        }
        
        assertEquals("Got: hello", processResult(Success("hello")))
        assertEquals("Error: failed", processResult(Error("failed")))
    }

    @Test
    fun testJavaInteroperability() {
        // Test that our Kotlin code can be called from Java-style code
        val duration = Duration.of(5, ChronoUnit.MINUTES)
        assertNotNull(duration)
        assertEquals(5 * 60, duration.seconds)
    }
}