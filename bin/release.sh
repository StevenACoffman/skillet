#!/bin/bash
# this script will update to the release version
# and make a new branch
gh-find-latest() {
  local owner=$1 project=$2
  local release_url=$(curl -Ls -o /dev/null -w '%{url_effective}' "https://github.com/${owner}/${project}/releases/latest")
  export release_tag=$(basename $release_url)
}

# Get Release tag
gh-find-latest StevenACoffman skillet
echo "Latest Release is ${release_tag}"


if ! [ $# -eq 1 ] ; then
    echo "usage: ./bin/release [version]"
    exit 1
fi

VERSION=$1

if ! git diff-index --quiet HEAD -- ; then
    echo "uncommitted changes on HEAD, aborting"
    exit 1
fi

if [[ ${VERSION:0:1} != "v" ]] ; then
    echo "version strings must start with v"
    exit 1
fi

git fetch origin
git checkout origin/master

git tag -s "${VERSION}" -m "${VERSION}"
git push origin "${VERSION}"
git push origin HEAD:master

git pull


echo "Now go write some release notes! https://github.com/StevenACoffman/skillet/releases"
sleep 1
curl -sS "https://proxy.golang.org/github.com/\!steven\!a\!coffman/skillet/\@v/${VERSION}.info"