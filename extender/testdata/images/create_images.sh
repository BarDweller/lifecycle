echo ">>>>>>>>>> Building build base image..."

cat <<EOF >Dockerfile
FROM cnbs/sample-builder:bionic
COPY ./lifecycle /cnb/lifecycle
EOF
docker build -t $REGISTRY_HOST/test-builder .
docker push $REGISTRY_HOST/test-builder

RUN_IMAGE=cnbs/sample-stack-run:bionic