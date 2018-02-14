In order to deploy on openshift you need to give cluster-admin credentials to the service account used by perceiver so that it can list pods and images.  To do that run:

oc adm policy add-cluster-role-to-user cluster-admin system:serviceaccount:<project>:openshift-perceiver
