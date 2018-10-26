# Build Binary Notes

## Table of Contents

<!-- TOC -->

- [Build Binary Notes](#build-binary-notes)
    - [Table of Contents](#table-of-contents)
    - [Test Download Local Darwin](#test-download-local-darwin)
    - [Test Build Local Darwin](#test-build-local-darwin)
    - [Delete Github Releases](#delete-github-releases)

<!-- /TOC -->

---

## Test Download Local Darwin

To verify SHA, use `shasum` for darwin and `sha256sum` for linux.

```sh
export GIT_TAG="0.7.3-37-bc43358"
export OS="darwin"
export ARCH="amd64"
export FILENAME="envconsul_${GIT_TAG}_${OS}-${ARCH}.tgz"
export CMD="shasum"
```

```sh
curl --silent --location --output ${FILENAME} \
  --url "https://github.com/the-container-store/envconsul/releases/download/${GIT_TAG}/${FILENAME}" && \
curl --silent --location --output ${FILENAME}.sha256 \
  --url "https://github.com/the-container-store/envconsul/releases/download/${GIT_TAG}/${FILENAME}.sha256" && \
${CMD} -c ${FILENAME}.sha256 && \
tar -xvzf ${FILENAME} && \
ls -alh ./envconsul && \
chmod +x ./envconsul && \
./envconsul --version
```

>     envconsul_0.7.3-37-bc43358_darwin-amd64.tgz: OK
>     x envconsul
>     -rwxr-xr-x  1 45950  TCSD001\Domain Users   7.9M Oct 26 11:14 ./envconsul
>     envconsul v0.7.3 (0.7.3-37-bc43358)

```sh
unset GIT_TAG
unset OS
unset ARCH
unset FILENAME
unset CMD
rm envconsul*
```

## Test Build Local Darwin

```sh
docker build --build-arg OS=darwin --tag test/envconsul:darwin-local .
CONTAINER_ID=$(docker create test/envconsul:darwin-local)
docker cp ${CONTAINER_ID}:/envconsul envconsul
docker rm -v ${CONTAINER_ID}
./envconsul --version
rm envconsul
docker rmi test/envconsul:darwin-local
```

>     Successfully tagged test/envconsul:darwin-local
>     044a2904a7905610d128719b79b7bdf0f8e56cfc9352a905717fb61f9c4b4eb7
>     envconsul v0.7.3 (local)

## Delete Github Releases

Delete Release by ID: `DELETE /repos/:owner/:repo/releases/:release_id`

Obtain GIT_USER and GIT_TOKEN credentials from TCS VAULT.

```sh
export GIT_USER=
export GIT_TOKEN=
export GIT_OWNER="the-container-store"
export GIT_REPO="envconsul"
export GIT_TAG="0.7.3-37-bc43358"
```

```sh
git config user.name "${GIT_USER}"
export GIT_RELEASE=$(curl --silent --request GET \
  --header "Authorization: token ${GIT_TOKEN}" \
  --url "https://api.github.com/repos/${GIT_OWNER}/${GIT_REPO}/releases/tags/${GIT_TAG} \
  | jq -r '.id')"
echo "GIT_RELEASE=${GIT_RELEASE}"
```

```sh
curl --silent --request DELETE \
  --header "Authorization: token ${GIT_TOKEN}" \
  --url "https://api.github.com/repos/${GIT_OWNER}/${GIT_REPO}/releases/${GIT_RELEASE}"
```

```sh
git config user.name ""
unset GIT_TOKEN
unset GIT_OWNER
unset GIT_REPO
unset GIT_TAG
unset GIT_RELEASE
```
