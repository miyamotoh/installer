package powervs

import (
	"context"
	"fmt"

	"github.com/openshift/installer/pkg/types"
	powervstypes "github.com/openshift/installer/pkg/types/powervs"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Validate executes platform specific validation/
func Validate(client API, config *types.InstallConfig) error {
	allErrs := field.ErrorList{}

	if config.Platform.PowerVS == nil {
		allErrs = append(allErrs, field.Required(field.NewPath("platform", "powervs"), "Power VS Validation requires a Power VS platform configuration."))
	} else {
		if config.ControlPlane != nil {
			fldPath := field.NewPath("controlPlane")
			allErrs = append(allErrs, validateMachinePool(fldPath, config.ControlPlane)...)
		}
		for idx, compute := range config.Compute {
			fldPath := field.NewPath("compute").Index(idx)
			allErrs = append(allErrs, validateMachinePool(fldPath, &compute)...)
		}
		fldPath := field.NewPath("VPC")
		allErrs = append(allErrs, validateExistingVPC(fldPath, client, config)...)
	}
	return allErrs.ToAggregate()
}

func validateMachinePool(fldPath *field.Path, machinePool *types.MachinePool) field.ErrorList {
	allErrs := field.ErrorList{}
	if machinePool.Architecture != "ppc64le" {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("architecture"), machinePool.Architecture, []string{"ppc64le"}))
	}
	return allErrs
}

// ValidatePreExistingDNS ensures no pre-existing DNS record exists in the CIS
// DNS zone or IBM DNS zone for cluster's Kubernetes API.
func ValidatePreExistingDNS(client API, ic *types.InstallConfig, metadata MetadataAPI) error {
	allErrs := field.ErrorList{}

	fldPath := field.NewPath("baseDomain")
	if ic.Publish == types.ExternalPublishingStrategy {
		allErrs = append(allErrs, validatePreExistingPublicDNS(fldPath, client, ic, metadata)...)
	} else {
		allErrs = append(allErrs, validatePreExistingPrivateDNS(fldPath, client, ic, metadata)...)
	}

	return allErrs.ToAggregate()
}

func validatePreExistingPublicDNS(fldPath *field.Path, client API, ic *types.InstallConfig, metadata MetadataAPI) field.ErrorList {
	allErrs := field.ErrorList{}
	// Get CIS CRN
	crn, err := metadata.CISInstanceCRN(context.TODO())
	if err != nil {
		return append(allErrs, field.InternalError(fldPath, err))
	}

	// Get CIS zone ID by name
	zoneID, err := client.GetDNSZoneIDByName(context.TODO(), ic.BaseDomain, types.ExternalPublishingStrategy)
	if err != nil {
		return append(allErrs, field.InternalError(fldPath, err))
	}

	// Search for existing records
	recordNames := [...]string{fmt.Sprintf("api.%s", ic.ClusterDomain()), fmt.Sprintf("api-int.%s", ic.ClusterDomain())}
	for _, recordName := range recordNames {
		records, err := client.GetDNSRecordsByName(context.TODO(), crn, zoneID, recordName, types.ExternalPublishingStrategy)
		if err != nil {
			allErrs = append(allErrs, field.InternalError(fldPath, err))
		}

		// DNS record exists
		if len(records) != 0 {
			allErrs = append(allErrs, field.Duplicate(fldPath, fmt.Sprintf("record %s already exists in CIS zone (%s) and might be in use by another cluster, please remove it to continue", recordName, zoneID)))
		}
	}
	return allErrs
}

func validatePreExistingPrivateDNS(fldPath *field.Path, client API, ic *types.InstallConfig, metadata MetadataAPI) field.ErrorList {
	allErrs := field.ErrorList{}
	// Get DNS CRN
	crn, err := metadata.DNSInstanceCRN(context.TODO())
	if err != nil {
		return append(allErrs, field.InternalError(fldPath, err))
	}

	// Get CIS zone ID by name
	zoneID, err := client.GetDNSZoneIDByName(context.TODO(), ic.BaseDomain, types.InternalPublishingStrategy)
	if err != nil {
		return append(allErrs, field.InternalError(fldPath, err))
	}

	// Search for existing records
	recordNames := [...]string{fmt.Sprintf("api-int.%s", ic.ClusterDomain())}
	for _, recordName := range recordNames {
		records, err := client.GetDNSRecordsByName(context.TODO(), crn, zoneID, recordName, types.InternalPublishingStrategy)
		if err != nil {
			allErrs = append(allErrs, field.InternalError(fldPath, err))
		}

		// DNS record exists
		if len(records) != 0 {
			allErrs = append(allErrs, field.Duplicate(fldPath, fmt.Sprintf("record %s already exists in DNS zone (%s) and might be in use by another cluster, please remove it to continue", recordName, zoneID)))
		}
	}
	return allErrs
}

func validateExistingVPC(path *field.Path, client API, ic *types.InstallConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	var vpcRegion string
	var err error

	if ic.PowerVS.VPCZone == "" {
		if ic.PowerVS.VPCName != "" {
			vpcRegion, err = powervstypes.VPCRegionForPowerVSRegion(ic.PowerVS.Region)
			allErrs = append(allErrs, findVPCinRegion(client, ic.PowerVS.VPCName, vpcRegion, path)...)
		}
	} else {
		vpcRegion, err = powervstypes.VPCRegionForVPCZone(ic.PowerVS.VPCZone)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(path.Child("vpcZone"), nil, err.Error()))
		} else {
			allErrs = append(allErrs, findVPCinRegion(client, ic.PowerVS.VPCName, vpcRegion, path)...)
			allErrs = append(allErrs, findSubnetinRegion(client, ic.PowerVS.VPCSubnets, vpcRegion, path)...)
		}
	}

	return allErrs
}

func findVPCinRegion(client API, name string, region string, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if name == "" {
		return allErrs
	}

	vpcs, err := client.GetVPCs(context.TODO(), region)
	if err != nil {
		return append(allErrs, field.InternalError(path.Child("vpcName"), err))
	}

	found := false
	for _, vpc := range vpcs {
		if *vpc.Name == name {
			found = true
			break
		}
	}
	if !found {
		allErrs = append(allErrs, field.NotFound(path.Child("vpcName"), name))
	}

	return allErrs
}

func findSubnetinRegion(client API, subnets []string, region string, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(subnets) == 0 {
		return allErrs
	}

	_, err := client.GetSubnetByName(context.TODO(), subnets[0], region)
	if err != nil {
		return append(allErrs, field.InternalError(path.Child("vpcSubnets"), err))
	}

	return allErrs
}
