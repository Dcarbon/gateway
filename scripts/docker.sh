TAG=dcarbon/gateway:hackathon_v1

docker build -t $TAG .
if [[ "$1" == "push" ]];then
    docker push $TAG
fi
