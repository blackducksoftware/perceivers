#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o quay-perceiver github.com/blackducksoftware/perceivers/cmd/quay-perceiver

docker build -t gautambaghel/quay:latest .

docker push gautambaghel/quay:latest

podname=$(kubectl get pods -n bd-ops --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep "quay")

kubectl delete pod -n bd-ops $podname

podname=$(kubectl get pods -n bd-ops --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep "quay")

sleep 5

kubectl logs -f -n bd-ops $podname
