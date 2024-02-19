TAG=dcarbon/gateway

docker build -t $TAG .
if [[ "$1" == "push" ]];then
    docker push $TAG
fi
