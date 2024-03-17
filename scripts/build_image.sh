echo "start build image"
docker build -t dev.registry.cyprinus.sg.cyprinus.tech/example:dev --build-arg GIT_TOKEN=$GIT_TOKEN .
echo "push image"
docker push dev.registry.cyprinus.sg.cyprinus.tech/example:dev
