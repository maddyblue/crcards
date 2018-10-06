set -ex

IMG=gcr.io/hots-cockroach/crcards:latest

go generate
go build -o crcards
docker build -t $IMG .
docker push $IMG
kubectl get po | grep crcards | awk '{print $1}' | xargs kubectl delete po
