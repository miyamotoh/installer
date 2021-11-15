package powervs

import (
	"github.com/openshift/installer/pkg/terraform"
	"github.com/openshift/installer/pkg/terraform/stages"
)

// PlatformStages are the stages to run to provision the infrastructure in AWS.
var PlatformStages = []terraform.Stage{
	stages.NewStage("powervs", "bootstrap", stages.WithNormalDestroy()),
	stages.NewStage("powervs", "master", stages.WithNormalDestroy()),
	stages.NewStage("powervs", "loadbalancer"),
	stages.NewStage("powervs", "dns"),
}
