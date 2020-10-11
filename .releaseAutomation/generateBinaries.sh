#!/bin/bash
dockerhubRepo="masteralt/requestdebugger"
releaseVersion=$(git rev-parse --abbrev-ref HEAD)
osTypes=( "darwin-amd64" "linux-386" "linux-amd64" "linux-arm" "linux-arm64" "windows-386" "windows-amd64" )

releaseDirectory="$(git rev-parse --show-toplevel)/RequestDebuggerBinariesForAllOS/${releaseVersion}"

# Create a zip directory of older version to save space.
if [ -d ${releaseDirectory} ] || [ -f "${releaseDirectory}.zip" ];
then
  echo "Release Binary directory or zip file already exist"
else
  ls "$(git rev-parse --show-toplevel)/RequestDebuggerBinariesForAllOS/" | grep -v zip | xargs -I % sh -c "zip -r %.zip %; rm -rf %"
  mkdir -p ${releaseDirectory}
fi

# Generate golang Binaries for All the available OS Types
for osIterator in ${osTypes[@]};
do
  os=$(echo $osIterator | cut -d "-" -f1)
  arch=$(echo $osIterator | cut -d "-" -f2)
  echo "Building binary for os ${os} and architecture ${arch} and storing it in ${releaseDirectory}/"
  CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -o "${releaseDirectory}/requestDebugger-${os}-${arch}" $(git rev-parse --show-toplevel)/requestHeadersQueryParamsAndBody.go
  chmod +x "${releaseDirectory}/requestDebugger-${os}-${arch}"
done

# create a zip out of the directory and wipe out the binary files
zip -r ${releaseDirectory} ${releaseDirectory}
rm -rf ${releaseDirectory}

# Build docker images and push to docker registry
echo "Building docker image for the version ${releaseVersion}"
docker build -t ${dockerhubRepo}:${releaseVersion} $(git rev-parse --show-toplevel)
docker tag ${dockerhubRepo}:${releaseVersion} ${dockerhubRepo}:latest
docker push ${dockerhubRepo}:${releaseVersion}
docker push ${dockerhubRepo}:latest
