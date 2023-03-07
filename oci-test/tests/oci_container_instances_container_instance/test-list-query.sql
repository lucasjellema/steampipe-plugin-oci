select display_name, id, lifecycle_state
from oci.oci_containerengine_clusteroci_container_instances_container_instance
where display_name = '{{ resourceName }}';