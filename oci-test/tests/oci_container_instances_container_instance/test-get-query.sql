select display_name, id, lifecycle_state
from oci.oci_container_instances_container_instance
where id = '{{ output.resource_id.value }}';