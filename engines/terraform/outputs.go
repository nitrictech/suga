package terraform

import (
	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

func (td *TerraformDeployment) createOutputs() {
	for name, module := range td.terraformResources {
		td.createOutputsForModule(name, module)
	}

	for name, module := range td.terraformInfraResources {
		td.createOutputsForModule(name, module)
	}
}

func (td *TerraformDeployment) createOutputsForModule(name string, module cdktf.TerraformHclModule) {
	cdktf.NewTerraformOutput(td.stack, jsii.Sprintf("%s_outputs", name), &cdktf.TerraformOutputConfig{
		Value: module,
	})
}
