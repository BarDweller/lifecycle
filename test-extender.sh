set -e

LIFECYCLE_REPO_PATH=$PWD
MAGENTA="\e[35m"
RED="\e[91m"
RESET="\e[0m"

echo -e "$MAGENTA>>>>>>>>>> Preparing registry...$RESET"

if [ -z "$REGISTRY_HOST" ]; then
  REGISTRY_HOST="localhost:5000"
fi
echo "REGISTRY_HOST: $REGISTRY_HOST"

if [ -z "$DEBUG" ]; then 
  DEBUG="info"
fi

if [ -z "$TESTDATA" ]; then
  TESTDATA="testdata"
fi
echo "TESTDATA: $TESTDATA"
FIXTURES_PATH=$LIFECYCLE_REPO_PATH/extender/$TESTDATA
if [ ! -d $FIXTURES_PATH ]; then
  echo -e "$RED Error: TESTDATA env var references missing path $FIXTURES_PATH $RESET"
  exit
fi

echo -e "$MAGENTA>>>>>>>>>> Cleanup old images$RESET"

# Remove output images from daemon - note that they STILL EXIST in the local registry
docker image rm $REGISTRY_HOST/test-builder --force
docker image rm $REGISTRY_HOST/extended/runimage --force   # run image to extend
docker image rm $REGISTRY_HOST/appimage --force

echo -e "$MAGENTA>>>>>>>>>> Building lifecycle...$RESET"

make clean build-linux-amd64
cd $LIFECYCLE_REPO_PATH/out/linux-amd64

echo -e "$MAGENTA>>>>>>>>>> Create images$RESET"

source $LIFECYCLE_REPO_PATH/extender/$TESTDATA/images/create_images.sh
echo "RUN_IMAGE: $RUN_IMAGE"

echo -e "$MAGENTA>>>>>>>>>> Building extender minimal image...$RESET"

cat <<EOF >Dockerfile.extender
FROM gcr.io/distroless/static
COPY ./lifecycle /cnb/lifecycle
CMD /cnb/lifecycle/extender
ENV CNB_USER_ID=1000
ENV CNB_GROUP_ID=1000
ENV CNB_STACK_ID="dummy.extender.stack.id"
EOF
docker build -f Dockerfile.extender -t $REGISTRY_HOST/extender .
docker push $REGISTRY_HOST/extender

echo -e "$MAGENTA>>>>>>>>>> Preparing fixtures...$RESET"
cd $FIXTURES_PATH

rm -rf ./kaniko
mkdir -p ./kaniko
rm -rf ./kaniko-run
mkdir -p ./kaniko-run


if [ -d workspace-orig ]; then
  echo -e "$MAGENTA>>>>>>>>>> Cloning workspace ...$RESET"
  if [ -d workspace ]; then
    rm -rf workspace
  fi
  cp -rp workspace-orig workspace
fi

echo -e "$MAGENTA>>>>>>>>>> Running detect...$RESET"

docker run \
  -v $PWD/cnb/buildpacks/:/cnb/buildpacks \
  -v $PWD/cnb/ext/:/cnb/ext \
  -v $PWD/layers/:/layers \
  -v $PWD/platform/:/platform \
  -v $PWD/workspace/:/workspace \
  --user 1000:1000 \
  $REGISTRY_HOST/test-builder \
  /cnb/lifecycle/detector -order /layers/order.toml -log-level $DEBUG

echo -e "$MAGENTA>>>>>>>>>> Running build for extensions...$RESET"

docker run \
  -v $PWD/cnb/buildpacks/:/cnb/buildpacks \
  -v $PWD/cnb/ext/:/cnb/ext \
  -v $PWD/layers/:/layers \
  -v $PWD/platform/:/platform \
  -v $PWD/workspace/:/workspace \
  --user 1000:1000 \
  $REGISTRY_HOST/test-builder \
  /cnb/lifecycle/builder -use-extensions -log-level $DEBUG

# Copy output /layers/config/metadata.toml so that the run extender can use it
# (otherwise it will be overwritten when running build for buildpacks)
cp ./layers/config/metadata.toml ./layers/config/extend-metadata.toml

echo -e "$MAGENTA>>>>>>>>>> Running extend on build image followed by build for buildpacks...$RESET"

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
  -log-level $DEBUG \
  -work-dir /kaniko \
  $REGISTRY_HOST/test-builder \
  $REGISTRY_HOST/extended/buildimage

docker pull $REGISTRY_HOST/extended/buildimage

echo -e "$MAGENTA>>>>>>>>>> Running extend on run image... $RUN_IMAGE $RESET"

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
  -log-level $DEBUG \
  -work-dir /kaniko \
  $RUN_IMAGE \
  $REGISTRY_HOST/extended/runimage

docker pull $REGISTRY_HOST/extended/runimage

echo -e "$MAGENTA>>>>>>>>>> Exporting final app image...$RESET"

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

echo -e "$MAGENTA>>>>>>>>>> Validate app imagei $RESET"

source $LIFECYCLE_REPO_PATH/extender/$TESTDATA/images/validate_run_image.sh

