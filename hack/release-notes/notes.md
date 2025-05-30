Welcome to ${TAG}! :tada:

<!-- In this release, we:
* Summary of high-level changes -->

<!-- Thanks to:
* @${GITHUB_USER} -->

<details>
<summary>Run via docker</summary>

```bash
# via ghcr.io
docker run --name trickster -d -v /path/to/trickster.yaml:/etc/trickster/trickster.yaml -p 0.0.0.0:8480:8480 ghcr.io/trickstercache/trickster:${TAG}

# via docker.io
docker run --name trickster -d -v /path/to/trickster.yaml:/etc/trickster/trickster.yaml -p 0.0.0.0:8480:8480 docker.io/trickstercache/trickster:${TAG}
```
</details>

<details>
<summary>Run via kubernetes/helm</summary>

```bash
helm install trickster oci://ghcr.io/trickstercache/charts/trickster --version ${TAG}
```

For more information, see the [helm chart](https://github.com/trickstercache/helm-charts).
</details>
