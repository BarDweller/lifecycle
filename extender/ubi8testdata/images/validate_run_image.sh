echo ">>>>>>>>>> Image available for testing"
echo "  Image available at : $REGISTRY_HOST/appimage"
echo "  Launch via docker with args as appropriate, eg."
echo "    docker run --rm --entrypoint curl -it $REGISTRY_HOST/appimage google.com"

echo ">cat /opt/arg.txt"
docker run --rm --entrypoint cat -it $REGISTRY_HOST/appimage /opt/arg.txt
echo ">curl google.com"
docker run --rm --entrypoint curl -it $REGISTRY_HOST/appimage google.com