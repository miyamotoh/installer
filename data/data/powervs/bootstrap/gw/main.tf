resource "ibm_is_public_gateway" "dns_vm_gateway" {
  count = var.gateway_attached ? 0 : 1
  name  = "${var.cluster_id}-gateway"
  vpc   = var.vpc_id
  zone  = var.vpc_zone
}

resource "ibm_is_subnet_public_gateway_attachment" "subnet_public_gateway_attachment" {
  count          = var.gateway_attached ? 0 : 1
  subnet         = var.vpc_subnet_id
  public_gateway = ibm_is_public_gateway.dns_vm_gateway[0].id
}
