set -e

LIFECYCLE_REPO_PATH=$PWD

echo ">>>>>>>>>> Preparing registry..."

if [ -z "$REGISTRY_HOST" ]; then
  REGISTRY_HOST="localhost:5000"
fi
echo "REGISTRY_HOST: $REGISTRY_HOST"

if [ -z "$TESTDATA" ]; then
  TESTDATA="testdata"
fi
echo "TESTDATA: $TESTDATA"
echo "RUN_IMAGE: $RUN_IMAGE"

echo ">>>>>>>>>> Cleanup old images"

# Remove output images from daemon - note that they STILL EXIST in the local registry
docker image rm $REGISTRY_HOST/test-builder --force
docker image rm $REGISTRY_HOST/extended/runimage --force   # run image to extend
docker image rm $REGISTRY_HOST/appimage --force

echo ">>>>>>>>>> Building lifecycle..."

make clean build-linux-amd64
cd $LIFECYCLE_REPO_PATH/out/linux-amd64

echo ">>>>>>>>>> Create images"

source $LIFECYCLE_REPO_PATH/extender/$TESTDATA/images/create_images.sh

echo ">>>>>>>>>> Building extender minimal image..."

cat <<EOF >Dockerfile.extender
FROM gcr.io/distroless/static
COPY ./lifecycle /cnb/lifecycle
CMD /cnb/lifecycle/extender
EOF
docker build -f Dockerfile.extender -t $REGISTRY_HOST/extender .
docker push $REGISTRY_HOST/extender

echo ">>>>>>>>>> Preparing fixtures..."
FIXTURES_PATH=$LIFECYCLE_REPO_PATH/extender/$TESTDATA
cd $FIXTURES_PATH

rm -rf ./kaniko
mkdir -p ./kaniko
rm -rf ./kaniko-run
mkdir -p ./kaniko-run

echo ">>>>>>>>>> Running detect..."

docker run \
  -v $PWD/cnb/buildpacks/:/cnb/buildpacks \
  -v $PWD/cnb/ext/:/cnb/ext \
  -v $PWD/layers/:/layers \
  -v $PWD/platform/:/platform \
  -v $PWD/workspace/:/workspace \
  --user 1000:1000 \
  $REGISTRY_HOST/test-builder \
  /cnb/lifecycle/detector -order /layers/order.toml -log-level debug

echo ">>>>>>>>>> Running build for extensions..."

docker run \
  -v $PWD/cnb/buildpacks/:/cnb/buildpacks \
  -v $PWD/cnb/ext/:/cnb/ext \
  -v $PWD/layers/:/layers \
  -v $PWD/platform/:/platform \
  -v $PWD/workspace/:/workspace \
  --user 1000:1000 \
  $REGISTRY_HOST/test-builder \
  /cnb/lifecycle/builder -use-extensions -log-level debug

# Copy output /layers/config/metadata.toml so that the run extender can use it
# (otherwise it will be overwritten when running build for buildpacks)
cp ./layers/config/metadata.toml ./layers/config/extend-metadata.toml

echo ">>>>>>>>>> Running extend on build image followed by build for buildpacks..."

docker run \
  -v $PWD/cnb/buildpacks/:/cnb/buildpacks \
  -v $PWD/cnb/ext/:/cnb/ext \
  -v $PWD/kaniko/:/kaniko \
  -v $PWD/layers/:/layers \
  -v $PWD/workspace/:/workspace \
  -e REGISTRY_HOST=$REGISTRY_HOST \
  --user 0:0 \
  --network host \
  $REGISTRY_HOST/extender \
  /cnb/lifecycle/extender \
  -app /workspace \
  -cache-image $REGISTRY_HOST/extended/buildimage/cache \
  -config /layers/config/metadata.toml \
  -kind build \
  -log-level debug \
  -work-dir /kaniko \
  $REGISTRY_HOST/test-builder \
  $REGISTRY_HOST/extended/buildimage

docker pull $REGISTRY_HOST/extended/buildimage

echo ">>>>>>>>>> Running extend on run image... $RUN_IMAGE"

docker run \
  -v $PWD/cnb/ext/:/cnb/ext \
  -v $PWD/kaniko-run/:/kaniko \
  -v $PWD/layers/:/layers \
  -v $PWD/workspace/:/workspace \
  -e REGISTRY_HOST=$REGISTRY_HOST \
  --user 0:0 \
  --network host \
  $REGISTRY_HOST/extender \
  /cnb/lifecycle/extender \
  -app /workspace \
  -cache-image $REGISTRY_HOST/extended/runimage/cache \
  -config /layers/config/extend-metadata.toml \
  -kind run \
  -log-level debug \
  -work-dir /kaniko \
  $RUN_IMAGE \
  $REGISTRY_HOST/extended/runimage

docker pull $REGISTRY_HOST/extended/runimage

echo ">>>>>>>>>> Exporting final app image..."

docker run \
  -v $PWD/cnb/buildpacks/:/cnb/buildpacks \
  -v $PWD/cnb/ext/:/cnb/ext \
  -v $PWD/layers/:/layers \
  -v $PWD/platform/:/platform \
  -v $PWD/workspace/:/workspace \
  --user 0:0 \
  --network host \
  $REGISTRY_HOST/test-builder \
  /cnb/lifecycle/exporter -log-level debug -run-image $REGISTRY_HOST/extended/runimage $REGISTRY_HOST/appimage

docker pull $REGISTRY_HOST/appimage

echo ">>>>>>>>>> Validate app image"

source $LIFECYCLE_REPO_PATH/extender/$TESTDATA/images/validate_run_image.sh

