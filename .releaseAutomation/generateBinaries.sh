#!/bin/bash
dockerhubRepo="kingalt/requestdebugger"
releaseVersion=$(git rev-parse --abbrev-ref HEAD)
osTypes=( "aix-ppc64" "darwin-amd64" "dragonfly-amd64" "freebsd-386" "freebsd-amd64" "freebsd-arm" "illumos-amd64" "js-wasm" "linux-386" "linux-amd64" "linux-arm" "linux-arm64" "linux-ppc64" "linux-ppc64le" "linux-mips" "linux-mipsle" "linux-mips64" "linux-mips64le" "linux-s390x" "netbsd-386" "netbsd-amd64" "netbsd-arm" "openbsd-386" "openbsd-amd64" "openbsd-arm" "openbsd-arm64" "plan9-386" "plan9-amd64" "plan9-arm" "solaris-amd64" "windows-386" "windows-amd64" )

releaseDirectory="$(git rev-parse --show-toplevel)/RequestDebuggerBinariesForAllOS/${releaseVersion}"

# Create a zip directory of older version to save space.
if [ -d ${releaseDirectory} ];
then
  echo "Release Binary directory already exists"
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

# Build docker images and push to docker registry
echo "Building docker image for the version ${releaseVersion}"
docker build -t ${dockerhubRepo}:${releaseVersion} $(git rev-parse --show-toplevel)
docker tag ${dockerhubRepo}:${releaseVersion} ${dockerhubRepo}:latest
docker push ${dockerhubRepo}:${releaseVersion}
docker push ${dockerhubRepo}:latest
