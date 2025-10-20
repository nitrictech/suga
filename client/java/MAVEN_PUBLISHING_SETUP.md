# Maven Publishing Setup for Testing

## GPG Key Configuration ✅

### Created GPG Key for Maven Publishing
- **Key ID:** `65B6D9563A0F4E37`
- **Full Fingerprint:** `1481359CF8DB70B2BDF1B4E165B6D9563A0F4E37`
- **Purpose:** Maven Central test publishing
- **Expires:** January 18, 2026
- **Email:** sean@nitric.io

### Key Server Upload ✅
Key has been uploaded to `keyserver.ubuntu.com` and is publicly available for verification.

## Settings.xml Location Options

### 1. User Settings (Default Location)
```
%USERPROFILE%\.m2\settings.xml
```
**Path:** `C:\Users\{username}\.m2\settings.xml`

**Use Case:** Global settings for all Maven projects

### 2. Custom Settings File (Recommended for Testing)
```
client/java/test-settings.xml
```
**Usage:** `mvn deploy -s client/java/test-settings.xml`

**Use Case:** Project-specific settings for testing without affecting global configuration

### 3. Global Maven Settings
```
{MAVEN_HOME}\conf\settings.xml
```
**Use Case:** System-wide settings (not recommended for personal credentials)

## Test Publishing Commands

### 1. Dry Run (Verify Configuration)
```bash
cd client/java
mvn clean verify -P release -s test-settings.xml -DskipTests
```

### 2. Test Snapshot Publication
```bash
cd client/java
mvn versions:set -DnewVersion=0.0.1-TEST-SNAPSHOT
mvn clean deploy -P release -s test-settings.xml
```

### 3. Test Release Publication
```bash
cd client/java
mvn clean deploy -P release -s test-settings.xml
```

## Environment Variables Required

Set these before running publication commands:

```bash
# Windows PowerShell
$env:CENTRAL_USERNAME="your-username"
$env:CENTRAL_PASSWORD="your-token"
$env:GPG_PASSPHRASE="your-gpg-passphrase"

# Windows Command Prompt
set CENTRAL_USERNAME=your-username
set CENTRAL_PASSWORD=your-token
set GPG_PASSPHRASE=your-gpg-passphrase

# Git Bash / WSL
export CENTRAL_USERNAME="your-username"
export CENTRAL_PASSWORD="your-token"
export GPG_PASSPHRASE="your-gpg-passphrase"
```

## Security Best Practices ✅

1. **Sensitive Data Protection:**
   - Credentials use environment variables
   - `test-settings.xml` is included in git (safe template)
   - Personal settings files (`*settings.xml`) are gitignored

2. **GPG Key Security:**
   - Dedicated key for Maven publishing only
   - Key has expiration date (Jan 2026)
   - Passphrase required for signing

3. **Testing Isolation:**
   - Use custom settings file for testing
   - Separate from personal/global Maven settings

## Files Created

- ✅ `client/java/test-settings.xml` - Test configuration template
- ✅ `client/java/.gitignore` - Updated with settings file rules
- ✅ `client/java/MAVEN_PUBLISHING_SETUP.md` - This documentation

## Next Steps

1. **Set up Central Portal account** at https://central.sonatype.com/
2. **Configure environment variables** with your credentials
3. **Test publishing** using the commands above
4. **Verify artifacts** appear in Central Portal dashboard

## Troubleshooting

### GPG Issues
- Verify key exists: `gpg --list-secret-keys --keyid-format LONG`
- Test signing: `gpg --local-user 65B6D9563A0F4E37 --armor --detach-sign file.txt`

### Maven Issues
- Check settings: `mvn help:effective-settings -s test-settings.xml`
- Verbose output: `mvn clean deploy -P release -s test-settings.xml -X`

### Central Portal Issues
- Verify credentials in environment variables
- Check server ID matches in settings.xml (`central`)
- Ensure artifacts are properly signed