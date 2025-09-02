# Trusted Publishing Setup for PyPI

This document explains how to set up OpenID Connect (OIDC) trusted publishing for the suga-client Python package. Trusted publishing allows GitHub Actions to publish to PyPI without storing API tokens as secrets.

## How Trusted Publishing Works

1. **GitHub generates a short-lived OIDC token** during workflow execution
2. **PyPI validates the token** against configured trusted publishers  
3. **PyPI grants publishing permissions** if the token matches the configuration
4. **No long-lived API tokens** need to be stored or managed

## Benefits

- **More secure**: No long-lived secrets stored in GitHub
- **Easier management**: No need to rotate API tokens
- **Scoped access**: Tokens are automatically scoped to specific projects
- **Audit trail**: Better tracking of who published what

## Setup Instructions

### 1. Create PyPI Account and Project

1. Go to [https://pypi.org](https://pypi.org) and create an account if you don't have one
2. For testing, also create an account at [https://test.pypi.org](https://test.pypi.org)

### 2. Initial Package Upload

Since trusted publishing requires the project to exist first, you need to do an initial upload with an API token:

1. Go to **Account Settings** → **API Tokens** → **Add API token**
2. Create a token scoped to the specific project (or global if project doesn't exist yet)
3. Use this token for the first upload to create the project on PyPI

### 3. Configure Trusted Publishing on PyPI

#### For Production PyPI (pypi.org):

1. Go to [https://pypi.org](https://pypi.org) and log in
2. Navigate to your project: `https://pypi.org/project/suga-client/`
3. Click **"Manage"** → **"Publishing"**
4. In the **"Trusted publishers"** section, click **"Add a new publisher"**
5. Fill in the details:
   - **PyPI Project Name**: `suga-client`
   - **Owner**: `nitrictech`
   - **Repository name**: `suga`  
   - **Workflow filename**: `publish.yaml`
   - **Environment name**: (leave blank unless using GitHub environments)

#### For TestPyPI (test.pypi.org):

1. Go to [https://test.pypi.org](https://test.pypi.org) and log in
2. Navigate to your project: `https://test.pypi.org/project/suga-client/`
3. Click **"Manage"** → **"Publishing"**
4. Add a new trusted publisher with:
   - **PyPI Project Name**: `suga-client`
   - **Owner**: `nitrictech`
   - **Repository name**: `suga`
   - **Workflow filename**: `test-publish-python.yaml`
   - **Environment name**: (leave blank)

### 4. Workflow Configuration

The workflows are already configured correctly with:

```yaml
permissions:
  id-token: write    # Required for OIDC token
  contents: read     # Required to read repo

steps:
  - name: Publish to PyPI
    uses: pypa/gh-action-pypi-publish@release/v1
    with:
      packages-dir: ./client/python/dist/
      # No password field - uses OIDC instead
```

### 5. Testing the Setup

#### Test with TestPyPI:

1. Go to GitHub Actions in the repository
2. Run the **"Test Publish Python Client"** workflow manually
3. Check the logs to ensure publishing succeeds
4. Verify the package appears on [https://test.pypi.org/project/suga-client/](https://test.pypi.org/project/suga-client/)

#### Production Publishing:

1. Create a git tag (e.g., `v1.0.0`) to trigger the main publish workflow
2. The workflow will automatically publish to production PyPI

## Troubleshooting

### Common Issues:

1. **"missing or insufficient OIDC token permissions"**
   - Ensure `permissions: id-token: write` is set at the job level
   - Remove any `password:` fields from the publish step

2. **"Trusted publishing exchange failure"**  
   - Verify the trusted publisher configuration on PyPI matches exactly:
     - Repository owner: `nitrictech`
     - Repository name: `suga`
     - Workflow filename matches the actual file
   - Check that the project exists on PyPI

3. **"This filename has already been used"**
   - Use `skip-existing: true` in the workflow (already configured for TestPyPI)
   - For production, ensure you're incrementing version numbers

### Verification Commands:

Check what GitHub is sending:
```bash
# In the workflow, add this debug step:
- name: Debug OIDC Token  
  run: |
    echo "GITHUB_REPOSITORY: $GITHUB_REPOSITORY"
    echo "GITHUB_REF: $GITHUB_REF"
    echo "GITHUB_WORKFLOW: $GITHUB_WORKFLOW"
```

## Migration from API Tokens

If you were previously using API tokens:

1. Set up trusted publishing as described above
2. Test that it works with TestPyPI
3. Remove the API token secrets from GitHub repository settings
4. The old tokens can be revoked from PyPI account settings

## Security Considerations

- Trusted publishing tokens are automatically scoped to the specific project
- Tokens are short-lived (typically 10 minutes)
- GitHub's OIDC provider is managed by GitHub - no additional infrastructure needed
- Access is controlled by GitHub repository permissions

## References

- [PyPI Trusted Publishing Documentation](https://docs.pypi.org/trusted-publishers/adding-a-publisher/)
- [GitHub OIDC Documentation](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- [PyPI Blog Post on Trusted Publishing](https://blog.pypi.org/posts/2023-04-20-introducing-trusted-publishers/)