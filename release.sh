#!/usr/bin/env bash

set -eux

version=$1

if [ -z "$version" ]; then
  echo "No version passed! Example usage: ./release.sh 1.0.0"
  exit 1
fi

echo "Running tests..."
make test

echo "Update version..."
old_version=$(grep -o "[0-9]*\.[0-9]*\.[0-9]*" main.go)
sed -i.bak 's/Version = "'$old_version'"/Version = "'$version'"/' main.go
sed -i.bak 's/VERSION="'$old_version'"/VERSION="'$version'"/' download.sh
sed -i.bak 's/'$old_version'/'$version'/' README.md

echo "Mark version as released in changelog..."
today=$(date +'%Y-%m-%d')
sed -i.bak 's/Unreleased/Unreleased\
\
## '$version' ('$today')/' CHANGELOG.md

echo "Build binaries..."
make build

echo "Update repository..."
rm *.bak
git add main.go download.sh README.md CHANGELOG.md
git commit -m "Bump version to ${version}"
git tag --sign --message="v$version" --force "v$version"
git tag --sign --message="latest" --force latest


echo "v$version tagged."
echo "Now, run 'git push origin master && git push --tags --force' and publish the release on GitHub."
