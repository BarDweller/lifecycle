  echo ">>>>>>>>>> Validating app image..."

  docker run --rm --entrypoint cat -it $REGISTRY_HOST/appimage /opt/arg.txt
  docker run --rm --entrypoint curl -it $REGISTRY_HOST/appimage google.com