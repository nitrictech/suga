package devserver

import (
	"encoding/json"
	"fmt"

	"github.com/nitrictech/suga/cli/internal/api"
	"github.com/nitrictech/suga/cli/internal/build"
	"github.com/nitrictech/suga/cli/internal/version"
)

type SugaProjectBuild struct {
	apiClient   *api.SugaApiClient
	broadcast   BroadcastFunc
	builder     *build.BuilderService
	currentTeam string
}

type ProjectBuild struct {
	// Empty struct - target is now read from the project file
}

type ProjectBuildSuccess struct {
	StackPath string `json:"stackPath"`
}

type ProjectBuildError struct {
	Message string `json:"message"`
}

func (n *SugaProjectBuild) OnConnect(send SendFunc) {
	// No-op
}

func (n *SugaProjectBuild) OnMessage(message json.RawMessage) {
	var buildMessage Message[ProjectBuild]
	err := json.Unmarshal(message, &buildMessage)
	if err != nil {
		fmt.Println("Error unmarshalling message", err)
		return
	}

	// Not the right message type continue
	if buildMessage.Type != "buildMessage" {
		return
	}

	// In future, we could support plugin library replacements here
	opts := build.BuildOptions{}

	stackPath, err := n.builder.BuildProjectFromFile(version.ConfigFileName, n.currentTeam, opts)
	if err != nil {
		fmt.Println(err.Error())

		n.broadcast(Message[any]{
			Type: "buildError",
			Payload: ProjectBuildError{
				Message: err.Error(),
			},
		})
		return
	}

	n.broadcast(Message[any]{
		Type: "buildSuccess",
		Payload: ProjectBuildSuccess{
			StackPath: stackPath,
		},
	})
}

func NewProjectBuild(apiClient *api.SugaApiClient, builder *build.BuilderService, broadcast BroadcastFunc, currentTeam string) (*SugaProjectBuild, error) {
	buildServer := &SugaProjectBuild{
		apiClient:   apiClient,
		broadcast:   broadcast,
		builder:     builder,
		currentTeam: currentTeam,
	}

	return buildServer, nil
}
