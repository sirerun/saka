#!/usr/bin/env bash
# Apply the saka stack to the current kubectl context.
# Expects the saka image already available to the cluster
# (kind load docker-image saka:local / mirror / pull secret).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
IMAGE="${SAKA_IMAGE:-saka:local}"

cd "$ROOT"
# Rewrite the kustomize image tag if SAKA_IMAGE is set (name:tag).
name="${IMAGE%%:*}"
tag="${IMAGE##*:}"
if [[ "$IMAGE" == "$name" ]]; then
  tag=latest
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cp -R deploy/k8s/. "$tmp/"
cat >"$tmp/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: saka
resources:
  - namespace.yaml
  - searxng.yaml
  - saka.yaml
images:
  - name: saka
    newName: ${name}
    newTag: ${tag}
EOF

kubectl apply -k "$tmp"
kubectl -n saka rollout status deploy/searxng --timeout=180s
kubectl -n saka rollout status deploy/saka --timeout=180s
echo "applied: namespace/saka  image=${IMAGE}"
echo "local access: kubectl -n saka port-forward svc/saka 8080:8080"
