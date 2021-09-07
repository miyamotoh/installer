## To hold the network creation:
## - network in the Power VS instance
## - VPC network

## The default names for these should be
## pvs-net-$cluster_id
## vpc-$cluster_id
## or similiar

## Then these can be passed through to the machine-config

## These should also be optional arguments in the install-config (e.g. Platform)
## so that users may specify them. Have them be "hidden" in that the survey doesn't ask for them
## unless the OCP leads disagree.

