TAG=dcarbon/gateway:prod5

docker build -t $TAG .
if [[ "$1" == "push" ]];then
    docker push $TAG
fi
