package terraform

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

type libraryID string

func NewLibraryID(team string, library string) libraryID {
	return libraryID(fmt.Sprintf("%s/%s", team, library))
}

func (id libraryID) Team() string {
	parts := strings.Split(string(id), "/")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func (id libraryID) Name() string {
	parts := strings.Split(string(id), "/")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

type libraryVersion string

type PlatformSpec struct {
	Name string `json:"name" yaml:"name"`

	Libraries map[libraryID]libraryVersion `json:"libraries" yaml:"libraries"`

	Variables map[string]Variable `json:"variables" yaml:"variables,omitempty"`

	ServiceBlueprints    map[string]*ServiceBlueprint  `json:"services" yaml:"services"`
	BucketBlueprints     map[string]*ResourceBlueprint `json:"buckets,omitempty" yaml:"buckets,omitempty"`
	TopicBlueprints      map[string]*ResourceBlueprint `json:"topics,omitempty" yaml:"topics,omitempty"`
	DatabaseBlueprints   map[string]*ResourceBlueprint `json:"databases,omitempty" yaml:"databases,omitempty"`
	EntrypointBlueprints map[string]*ResourceBlueprint `json:"entrypoints" yaml:"entrypoints"`
	InfraSpecs           map[string]*ResourceBlueprint `json:"infra" yaml:"infra"`
}

type Variable struct {
	Type        string      `json:"type" yaml:"type"`
	Description string      `json:"description" yaml:"description"`
	Default     interface{} `json:"default" yaml:"default"`
	Nullable    bool        `json:"nullable" yaml:"nullable"`
}

type Library struct {
	Team    string `json:"team" yaml:"team"`
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}

type Plugin struct {
	Library Library `json:"library" yaml:"library"`
	Name    string  `json:"name" yaml:"name"`
}

func (p PlatformSpec) GetLibrary(id libraryID) (*Library, error) {
	libVersion, ok := p.Libraries[id]
	if !ok {
		return nil, fmt.Errorf("library %s not found in platform spec, configured libraries in platform are: %v", id, slices.Collect(maps.Keys(p.Libraries)))
	}

	// pattern := `^(?P<team>[^/]+)/(?P<library>[^@]+)@(?P<version>.+)$`
	// re := regexp.MustCompile(pattern)

	// matches := re.FindStringSubmatch(string(libVersion))
	// if len(matches) == 0 {
	// 	return nil, fmt.Errorf("invalid library format: %s, expected format: `<team>/<platform>@<version>`", libVersion)
	// }

	// team := matches[re.SubexpIndex("team")]
	// libName := matches[re.SubexpIndex("library")]
	// version := matches[re.SubexpIndex("version")]

	return &Library{Team: id.Team(), Name: id.Name(), Version: string(libVersion)}, nil
}

func (p PlatformSpec) GetLibraries() map[libraryID]*Library {
	libraries := map[libraryID]*Library{}
	for id := range p.Libraries {
		libraries[id], _ = p.GetLibrary(id)
	}
	return libraries
}

type MissingResourceBlueprintError struct {
	IntentType     string
	IntentSubType  string
	AvailableTypes []string
}

func (e MissingResourceBlueprintError) Error() string {
	return fmt.Sprintf("platform does not define a '%s' type for %ss, available types: %v", e.IntentSubType, e.IntentType, e.AvailableTypes)
}

func NewMissingResourceBlueprintError(intentType, intentSubType string, availableTypes []string) error {
	return MissingResourceBlueprintError{
		IntentType:     intentType,
		IntentSubType:  intentSubType,
		AvailableTypes: availableTypes,
	}
}

func (p PlatformSpec) GetServiceBlueprint(intentSubType string) (*ServiceBlueprint, error) {
	spec := p.ServiceBlueprints

	if intentSubType == "" {
		intentSubType = "default"
	}

	concreteSpec, ok := spec[intentSubType]
	if !ok || concreteSpec == nil {
		return nil, NewMissingResourceBlueprintError("service", intentSubType, slices.Collect(maps.Keys(spec)))
	}

	return concreteSpec, nil
}

func (p PlatformSpec) GetResourceBlueprintsForType(typ string) (map[string]*ResourceBlueprint, error) {
	var specs map[string]*ResourceBlueprint

	switch typ {
	case "service":
		specs = map[string]*ResourceBlueprint{}
		for name, blueprint := range p.ServiceBlueprints {
			specs[name] = blueprint.ResourceBlueprint
		}
	case "entrypoint":
		specs = p.EntrypointBlueprints
	case "bucket":
		specs = p.BucketBlueprints
	case "topic":
		specs = p.TopicBlueprints
	default:
		return nil, fmt.Errorf("failed to resolve resource blueprint, no type %s in platform spec", typ)
	}

	return specs, nil
}

func (p PlatformSpec) GetResourceBlueprint(intentType string, intentSubType string) (*ResourceBlueprint, error) {
	if intentSubType == "" {
		intentSubType = "default"
	}

	var spec *ResourceBlueprint
	var availableTypes []string
	switch intentType {
	case "service":
		availableTypes = slices.Collect(maps.Keys(p.ServiceBlueprints))
		if serviceBlueprint, ok := p.ServiceBlueprints[intentSubType]; ok {
			spec = serviceBlueprint.ResourceBlueprint
		}
	case "entrypoint":
		availableTypes = slices.Collect(maps.Keys(p.EntrypointBlueprints))
		spec = p.EntrypointBlueprints[intentSubType]
	case "bucket":
		availableTypes = slices.Collect(maps.Keys(p.BucketBlueprints))
		spec = p.BucketBlueprints[intentSubType]
	case "topic":
		availableTypes = slices.Collect(maps.Keys(p.TopicBlueprints))
		spec = p.TopicBlueprints[intentSubType]
	case "database":
		availableTypes = slices.Collect(maps.Keys(p.DatabaseBlueprints))
		spec = p.DatabaseBlueprints[intentSubType]
	default:
		return nil, fmt.Errorf("failed to resolve resource blueprint, no type %s known in platform spec", intentType)
	}

	if spec == nil {
		return nil, NewMissingResourceBlueprintError(intentType, intentSubType, availableTypes)
	}

	return spec, nil
}

func PlatformSpecFromReader(reader io.Reader) (*PlatformSpec, error) {
	var spec PlatformSpec

	byt, err := afero.ReadAll(reader)
	if err != nil {
		return &PlatformSpec{}, nil
	}

	err = yaml.Unmarshal(byt, &spec)

	return &spec, err
}

func PlatformSpecFromFile(fs afero.Fs, filePath string) (*PlatformSpec, error) {
	file, err := fs.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return &PlatformSpec{}, fmt.Errorf("failed to read platform spec file %s: %w", filePath, err)
	}

	return PlatformSpecFromReader(file)
}

type PlatformReferencePrefix string

const (
	PlatformReferencePrefix_File  = "file:"
	PlatformReferencePrefix_Https = "https://"
	PlatformReferencePrefix_Git   = "git+"
)

func PlatformFromId(fs afero.Fs, platformId string, repositories ...PlatformRepository) (*PlatformSpec, error) {
	if strings.HasPrefix(platformId, PlatformReferencePrefix_File) {
		return PlatformSpecFromFile(fs, strings.Replace(platformId, PlatformReferencePrefix_File, "", -1))
	} else if strings.HasPrefix(platformId, PlatformReferencePrefix_Https) || strings.HasPrefix(platformId, PlatformReferencePrefix_Git) {
		return nil, fmt.Errorf("platform %s is not supported yet", platformId)
	}

	for _, repository := range repositories {
		platform, err := repository.GetPlatform(platformId)
		if errors.Is(err, ErrUnauthenticated) {
			return nil, errors.Wrap(err, "unable to authenticate with platform repository, please make sure you are logged in with `suga login`")
		} else if errors.Is(err, ErrPlatformNotFound) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("an unknown error occurred while fetching platform %s from platform repository, please try again later: %w", platformId, err)
		}

		return platform, nil
	}

	// TODO: check for close matches and list available platforms
	return nil, fmt.Errorf("platform %s not found. If the platform exists in a different team, switch teams using `suga team <team-name>`", platformId)
}

type pluginSource struct {
	Library libraryID `json:"library" yaml:"library"`
	Plugin  string    `json:"plugin" yaml:"plugin"`
}

func NewPluginSource(library libraryID, plugin string) pluginSource {
	return pluginSource{Library: library, Plugin: plugin}
}

type ResourceBlueprint struct {
	Source     pluginSource           `json:"source" yaml:"source"`
	Properties map[string]interface{} `json:"properties" yaml:"properties"`
	DependsOn  []string               `json:"depends_on" yaml:"depends_on,omitempty"`
	Variables  map[string]Variable    `json:"variables" yaml:"variables,omitempty"`
}

func (r *ResourceBlueprint) ResolvePlugin(platform *PlatformSpec) (*Plugin, error) {
	if r == nil {
		return nil, fmt.Errorf("resource blueprint is nil, this indicates a malformed platform spec")
	}
	if r.Source.Library == "" {
		return nil, fmt.Errorf("no source library specified for resource blueprint, this indicates a malformed platform spec")
	}

	if r.Source.Plugin == "" {
		return nil, fmt.Errorf("no source plugin specified for resource blueprint, this indicates a malformed platform spec")
	}

	lib, err := platform.GetLibrary(r.Source.Library)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve library for plugin %s, %w", r.Source.Plugin, err)
	}

	return &Plugin{Library: *lib, Name: r.Source.Plugin}, nil
}

type IdentitiesBlueprint struct {
	Identities []ResourceBlueprint `json:"identities" yaml:"identities"`
}

func (i IdentitiesBlueprint) GetIdentities() []ResourceBlueprint {
	if i.Identities == nil {
		return []ResourceBlueprint{}
	}
	return i.Identities
}

type Identifiable interface {
	GetIdentity(string) (*ResourceBlueprint, error)
	GetIdentities() map[string]ResourceBlueprint
}

type ServiceBlueprint struct {
	*ResourceBlueprint   `json:",inline" yaml:",inline"`
	*IdentitiesBlueprint `json:",inline" yaml:",inline"`
}
