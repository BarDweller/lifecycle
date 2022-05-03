OUTPUT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )/build-artifacts
mkdir -p ${OUTPUT_DIR}

cat << EOF > ${OUTPUT_DIR}/builder.toml
description = "empty ubi8 builder image"

[lifecycle]
  version = "0.13.3"

[stack]
  id = "ubi8.minimal"
  build-image = "${REGISTRY_HOST}/ubi8-empty-build:minimal"
  run-image = "${REGISTRY_HOST}/ubi8-empty-run:minimal"

EOF

cat <<EOF > ${OUTPUT_DIR}/Dockerfile.run-image
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV CNB_USER_ID=1000
ENV CNB_GROUP_ID=1000
ENV CNB_STACK_ID="ubi8.minimal"
LABEL io.buildpacks.stack.id="ubi8.minimal"

RUN microdnf install --setopt=install_weak_deps=0 --setopt=tsflags=nodocs \
  shadow-utils && microdnf clean all && groupadd cnb --gid \${CNB_GROUP_ID} && \
  useradd --uid \${CNB_USER_ID} --gid \${CNB_GROUP_ID} -m -s /bin/bash cnb

USER \${CNB_USER_ID}:\${CNB_GROUP_ID}

EOF

cat <<EOF > ${OUTPUT_DIR}/Dockerfile.run-image-java-11
FROM registry.access.redhat.com/ubi8/openjdk-11-runtime:latest

ENV CNB_USER_ID=1000
ENV CNB_GROUP_ID=1000
ENV CNB_STACK_ID="ubi8.minimal"
LABEL io.buildpacks.stack.id="ubi8.minimal"

USER 0

RUN groupadd cnb --gid \${CNB_GROUP_ID} && \
  useradd --uid \${CNB_USER_ID} --gid \${CNB_GROUP_ID} -m -s /bin/bash cnb

USER \${CNB_USER_ID}:\${CNB_GROUP_ID}

EOF

cat <<EOF > ${OUTPUT_DIR}/Dockerfile.run-image-java-17
FROM registry.access.redhat.com/ubi8/openjdk-17-runtime:latest

ENV CNB_USER_ID=1000
ENV CNB_GROUP_ID=1000
ENV CNB_STACK_ID="ubi8.minimal"
LABEL io.buildpacks.stack.id="ubi8.minimal"

USER 0

RUN groupadd cnb --gid \${CNB_GROUP_ID} && \
  useradd --uid \${CNB_USER_ID} --gid \${CNB_GROUP_ID} -m -s /bin/bash cnb

USER \${CNB_USER_ID}:\${CNB_GROUP_ID}

EOF

cat <<EOF > ${OUTPUT_DIR}/Dockerfile.build-image
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV CNB_USER_ID=1000
ENV CNB_GROUP_ID=1000
ENV CNB_STACK_ID="ubi8.minimal"
LABEL io.buildpacks.stack.id="ubi8.minimal"

RUN microdnf install --setopt=install_weak_deps=0 --setopt=tsflags=nodocs \
  shadow-utils && microdnf clean all && groupadd cnb --gid \${CNB_GROUP_ID} && \
  useradd --uid \${CNB_USER_ID} --gid \${CNB_GROUP_ID} -m -s /bin/bash cnb

EOF

echo -n ">>>>>>>>>> Removing old build/run image..."
docker image rm $REGISTRY_HOST/ubi8-empty-build:minimal --force
docker image rm $REGISTRY_HOST/ubi8-empty-run:minimal --force
docker image rm $REGISTRY_HOST/ubi8-java-11-run:minimal --force
docker image rm $REGISTRY_HOST/ubi8-java-17-run:minimal --force
docker image rm $REGISTRY_HOST/test-builder:pack --force

echo ">>>>>>>>>> Building build base image..."
docker build . -t $REGISTRY_HOST/ubi8-empty-build:minimal -f ${OUTPUT_DIR}/Dockerfile.build-image 
echo ">>>>>>>>>> Building run base images..."
docker build . -t $REGISTRY_HOST/ubi8-empty-run:minimal -f ${OUTPUT_DIR}/Dockerfile.run-image
docker build . -t $REGISTRY_HOST/ubi8-java-11-run:minimal -f ${OUTPUT_DIR}/Dockerfile.run-image-java-11
docker build . -t $REGISTRY_HOST/ubi8-java-17-run:minimal -f ${OUTPUT_DIR}/Dockerfile.run-image-java-17

echo ">>>>>>>>>> Pack creating builder image..."
pack builder create $REGISTRY_HOST/test-builder:pack --config ${OUTPUT_DIR}/builder.toml

docker push $REGISTRY_HOST/ubi8-empty-build:minimal
docker push $REGISTRY_HOST/ubi8-empty-run:minimal
docker push $REGISTRY_HOST/ubi8-java-11-run:minimal
docker push $REGISTRY_HOST/ubi8-java-17-run:minimal
docker push $REGISTRY_HOST/test-builder:pack

cat <<EOF >${OUTPUT_DIR}/Dockerfile.withlifecycle
FROM $REGISTRY_HOST/test-builder:pack
COPY ./lifecycle /cnb/lifecycle
EOF
docker build . -t $REGISTRY_HOST/test-builder -f ${OUTPUT_DIR}/Dockerfile.withlifecycle
docker push $REGISTRY_HOST/test-builder

RUN_IMAGE=$REGISTRY_HOST/ubi8-empty-run:minimal
