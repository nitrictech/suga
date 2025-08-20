package client

// BucketWithPermissions extends ResourceNameNormalizer with permission information
type BucketWithPermissions struct {
	ResourceNameNormalizer
	Permissions []string
}

// HasPermission checks if a specific permission is available
func (b BucketWithPermissions) HasPermission(permission string) bool {
	for _, p := range b.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasRead checks if read permission is available
func (b BucketWithPermissions) HasRead() bool {
	return b.HasPermission("read")
}

// HasWrite checks if write permission is available
func (b BucketWithPermissions) HasWrite() bool {
	return b.HasPermission("write")
}

// HasDelete checks if delete permission is available
func (b BucketWithPermissions) HasDelete() bool {
	return b.HasPermission("delete")
}