variable "cluster_id" {
  type        = string
  description = "The ID created by the installer to uniquely identify the created cluster."
}

variable "vpc_id" { type = string }
variable "vpc_subnet_id" { type = string }
variable "vpc_zone" { type = string }
variable "gateway_attached" { type = bool }
