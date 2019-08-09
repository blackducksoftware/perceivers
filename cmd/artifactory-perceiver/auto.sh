#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o artifactory-perceiver github.com/blackducksoftware/perceivers/cmd/artifactory-perceiver

docker build -t gautambaghel/art:latest .

docker push gautambaghel/art:latest

podname=$(kubectl get pods -n bd-ops --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep "artifactory")

kubectl delete pod -n bd-ops $podname

podname=$(kubectl get pods -n bd-ops --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep "artifactory")

sleep 5

kubectl logs -f -n bd-ops $podname
