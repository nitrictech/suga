# Suga Python Client

A Python client library for interacting with Suga cloud resources.

## Installation

```bash
pip install suga-client
```

## Usage

```python
from suga import Bucket

# Create a bucket instance
bucket = Bucket(storage_client, "my-bucket")

# Write data
bucket.write("key", b"data")

# Read data  
data = bucket.read("key")

# List objects
keys = bucket.list("prefix")

# Get presigned URLs
download_url = bucket.get_download_url("key")
upload_url = bucket.get_upload_url("key")
```

## License

MPL-2.0