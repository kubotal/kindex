# kindex

**kindex** is a small HTTP service that connects to a Kubernetes cluster and serves a single HTML page listing **Ingress** hosts as clickable links. It is meant as a lightweight “index” of cluster entry points (for example behind your own authentication or network controls).

## What you get

- **Cluster-wide Ingress list** — reads `Ingress` objects in all namespaces and sorts links alphabetically by label.
- **Sensible default links** — if you do not add custom annotations, each Ingress uses the first rule host; the link text is the hostname segment before the first dot (for example `grafana` for `grafana.example.com`).
- **HTTPS vs HTTP** — the target URL uses `https` when the host appears under `spec.tls` or when common **TLS passthrough** annotations are present (`nginx.ingress.kubernetes.io/ssl-passthrough`, `ingress.kubernetes.io/ssl-passthrough`, `haproxy.org/ssl-passthrough`).
- **Optional custom links** — annotations prefixed with `kindex.kubotal.io/link` (see below).
- **Appearance** — `--mode dark` or `--mode light` for the web UI.
- **Cluster label on the page** — by default the name comes from your kubeconfig’s current context (cluster entry). You can override it with `--clusterName`.

The application version is exposed in the HTML as a `<meta name="kindex-version" content="…">` tag (visible in “View page source”), not as on-screen text.

## Requirements

- A Kubernetes cluster you can reach with a valid kubeconfig, **or** in-cluster credentials when running as a Pod.
- To list Ingresses everywhere, the identity you use needs **`get` / `list` / `watch`** on **`ingresses`** in API group **`networking.k8s.io`** at cluster scope (the Helm chart installs a `ClusterRole` for that).

## Install and run (binary)

Build from source (Go toolchain required):

```bash
make version build   # writes internal/global/version_.go and builds bin/kindex
./bin/kindex serve
```

Or with `go build` after generating the version file (see `make version`).

Start the server:

```bash
kindex serve
```

By default it listens on **`0.0.0.0:7788`**. Open `http://localhost:7788/` (or your host and port).

### Kubeconfig

Configuration is resolved in this order:

1. **`--kubeconfig` / `-k`** — explicit file path  
2. **`KUBECONFIG`** environment variable (standard multi-file behaviour)  
3. **`~/.kube/config`**

Inside a Pod, the usual **service account** token and cluster CA are used when no kubeconfig is mounted.

### Useful flags (`kindex serve`)

| Flag | Description |
|------|-------------|
| `--bindAddr`, `-a` | Listen address (default `0.0.0.0`) |
| `--bindPort`, `-p` | Listen port (default `7788`) |
| `--kubeconfig`, `-k` | Path to kubeconfig file |
| `--mode` | `dark` or `light` (default `dark`) |
| `--clusterName` | Override the cluster name shown on the page |
| `--tls`, `-t` | Serve HTTPS (requires `--certDir`, `--certName`, `--keyName`) |
| `--logMode` | `text` or `json` |
| `--logLevel`, `-l` | e.g. `INFO`, `DEBUG` |

Run `kindex serve --help` for the full list.

### Version

```bash
kindex version
kindex version --extended   # includes build timestamp
```

## Custom links with annotations

Annotation keys must be exactly **`kindex.kubotal.io/link`** or start with that prefix followed by **`.`** or **`-`** (for example `kindex.kubotal.io/link.docs`). **`/`** is not allowed in annotation keys.

The value has up to three segments separated by **`:`** (split with at most two separators):

1. **Display text** — if empty, the same short hostname label as the default case is used.  
2. **Path** — appended to the host (leading `/` added if missing).  
3. **Description** — short text after the link.

Skip an annotation entirely by setting its value to **`""`**; that key does not produce a row.

Multiple matching annotations on one Ingress produce multiple rows.

## Helm

A chart lives under **`helm/kindex/`**.

Typical install (adjust host, ingress class, and optional display name):

```bash
helm upgrade --install kindex ./helm/kindex \
  --namespace kindex --create-namespace \
  --set ingress.host=index.example.com \
  --set ingress.class=nginx \
  --set server.clusterName=my-cluster
```

Important values:

- **`ingress.host`** and **`ingress.class`** — required when `ingress.enabled` is true (the template validates this).
- **`rbac.create`** / **`serviceAccount.create`** — default **true**; grants cluster-wide read-only access to Ingresses and wires the Pod service account.
- **`server.mode`** — `dark` or `light`.
- **`server.tls`** — when true, mounts a certificate secret and enables server TLS (configure issuer / certificate resources as needed).

See **`helm/kindex/values.yaml`** for images, resources, node selectors, tolerations, and name overrides.

## Container image

The Makefile builds and pushes an image (default registry/tag in the Makefile). Example:

```bash
make docker
```

## License

Apache License 2.0 — see the headers in the repository.
