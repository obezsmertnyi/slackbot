## Встановлення Flux в Kubernetes кластер розгорнутий локально за допомогою kind.


<!-- BEGIN_TF_DOCS -->


## Providers

| Name | Version |
|------|---------|
| <a name="provider_flux"></a> [flux](#provider\_flux) | n/a |
| <a name="provider_github"></a> [github](#provider\_github) | n/a |
| <a name="provider_kind"></a> [kind](#provider\_kind) | n/a |
| <a name="provider_tls"></a> [tls](#provider\_tls) | n/a |

## Resources

| Name | Type |
|------|------|
| [flux_bootstrap_git.this](https://registry.terraform.io/providers/hashicorp/flux/latest/docs/resources/bootstrap_git) | resource |
| [github_repository_deploy_key.this](https://registry.terraform.io/providers/hashicorp/github/latest/docs/resources/repository_deploy_key) | resource |
| [kind_cluster.this](https://registry.terraform.io/providers/hashicorp/kind/latest/docs/resources/cluster) | resource |
| [tls_private_key.ecdsa-p384-key](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_config_path"></a> [config\_path](#input\_config\_path) | The path to the kubeconfig file | `string` | `"~/.kube/config"` | no |
| <a name="input_flux_github_repo"></a> [flux\_github\_repo](#input\_flux\_github\_repo) | The name flux repo | `string` | `"flux-gitops"` | no |
| <a name="input_github_owner"></a> [github\_owner](#input\_github\_owner) | The name GitHub owner | `string` | `"obezsmertnyi"` | no |
| <a name="input_github_token"></a> [github\_token](#input\_github\_token) | The token for the GitHub | `string` | n/a | yes |
<!-- END_TF_DOCS -->