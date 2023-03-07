variable "resource_name" {
  type        = string
  default     = "steampipetest20200125"
  description = "Name of the resource used throughout the test."
}

variable "config_file_profile" {
  type        = string
  default     = "DEFAULT"
  description = "OCI credentials profile used for the test. Default is to use the default profile."
}

variable "container_instance_shape_config_ocpus" {
  type        = string
  default     = "2.0"
  description = "The total number of OCPUs available to the instance."
}


variable "container_instance_shape" {
  type        = string
  default     = "CI.Standard.E4.Flex"
  description = " The shape of the Container Instance. The shape determines the resources available to the Container Instance."
}


variable "container_instance_containers_image_url" {
  type        = string
  default     = "nginx:1.23.3"
  description = "The container image information. Currently only support public docker registry. Can be either image name, e.g containerImage, image name with version, e.g containerImage:v1 or complete docker image Url e.g docker.io/library/containerImage:latest"
}


variable "container_instance_availability_domain" {
  type        = string
  default     = "AD-1"
  description = "Availability Domain where the ContainerInstance is running."
}


variable "tenancy_ocid" {
  type        = string
  default     = "ocid1.tenancy.oc1..aaaaaaaag7c7slwmlvsodyym662ixlsonnihko2igwpjwwe2egmlf3gg6okq"
  description = "OCID of your tenancy."
}

variable "region" {
  type        = string
  default     = "us-ashburn-1"
  description = "OCI region used for the test. Does not work with default region in config, so must be defined here."
}

provider "oci" {
  tenancy_ocid        = var.tenancy_ocid
  config_file_profile = var.config_file_profile
  region              = var.region
}

resource "oci_core_vcn" "named_test_resource" {
  compartment_id = var.tenancy_ocid
  cidr_block     = "10.0.0.0/16"
}

resource "oci_core_subnet" "named_test_resource" {
  cidr_block     = "10.0.0.0/16"
  compartment_id = var.tenancy_ocid
  vcn_id         = oci_core_vcn.named_test_resource.id
}

resource "oci_container_instances_container_instance" "named_test_resource" {
    #Required
    availability_domain = var.container_instance_availability_domain
    compartment_id = var.tenancy_ocid
    display_name = var.resource_name
    containers {
        #Required
        image_url = var.container_instance_containers_image_url
    }
    shape = var.container_instance_shape
    shape_config {
        #Required
        ocpus = var.container_instance_shape_config_ocpus
    }
    vnics {
        #Required
        subnet_id = oci_core_subnet.named_test_resource.id
    }
}

output "resource_name" {
  value = var.resource_name
}

output "tenancy_ocid" {
  value = var.tenancy_ocid
}

output "resource_id" {
  value = oci_container_instances_container_instance.named_test_resource.id
}

output "lifecycle_state" {
  value = oci_container_instances_container_instance.named_test_resource.state
}

