# E5 -- Helm Chart Deploy

Acceptance: `helm install saka deploy/helm/saka` deploys the same saka+SearXNG stack as the plain kubectl-apply path, verified in CI.
fidelity: executable

- [ ] T5.1 Scaffold a Helm chart under deploy/helm/saka templating the existing namespace/saka/searxng manifests  Owner: TBD  Est: 1.5h  verifies: [UC-026]  acc: [helm template deploy/helm/saka renders valid Kubernetes YAML for namespace, saka Deployment/Service, and searxng Deployment/Service]
- [ ] T5.2 Parameterize image repo/tag, resource limits, and SearXNG settings via values.yaml  Owner: TBD  Est: 1h  verifies: [UC-026]  deps: [T5.1]  acc: [helm template deploy/helm/saka --set image.tag=test renders the overridden tag into the saka Deployment]
- [ ] T5.3 Add a CI job (or extend k8s-smoke) that runs `helm install` against the kind cluster and the existing deploy/smoke.sh  Owner: TBD  Est: 1h  verifies: [UC-026]  deps: [T5.1, T5.2]  acc: [.github/workflows/ci.yml's helm-smoke job passes on a fresh kind cluster]
- [ ] T5.4 Update README.md's "Self-hosted stack" section to document the Helm install path alongside kubectl apply  Owner: TBD  Est: 30m  verifies: [UC-026]  deps: [T5.2]  acc: [README.md shows a helm install saka deploy/helm/saka command in the self-hosted section]
