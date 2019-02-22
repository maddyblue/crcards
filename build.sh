set -ex

IMG=gcr.io/cockroach-shared/directory-crdb-io:latest

go generate
go build -o crcards
docker build -t $IMG .
docker push $IMG
kubectl get po | grep directory | awk '{print $1}' | xargs kubectl delete po
