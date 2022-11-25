# cluster-machine-approver
The cluster-machine-approver validates and approves CSRs for nodes attempting to join the cluster.


cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1
kind: CertificateSigningRequest
metadata:
  name: new
  namespace: kube-system
spec:
  request: $(cat server.csr | base64 | tr -d '\n')
  signerName: example.com/serving
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF