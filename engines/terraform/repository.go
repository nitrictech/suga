package terraform

type PlatformRepository interface {
	// <team>/<platform>/<revision>
	GetPlatform(string) (*PlatformSpec, error)
}
