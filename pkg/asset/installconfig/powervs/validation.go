package powervs

import (
	types "github.com/openshift/installer/pkg/types/powervs"
)

// Just see if we can create an IBMPISession
// @TODO: Expand this to use the install config creds
func ValidateForProvisioning(ic *types.Platform) error {
	_, err := GetSession(ic)
	return err
}
