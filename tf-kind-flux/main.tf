provider "kind" {}

provider "github" {
  owner = var.github_owner
  token = var.github_token
}

provider "flux" {
  kubernetes = {
    config_path            = var.config_path
    host                   = kind_cluster.this.endpoint
    client_certificate     = kind_cluster.this.client_certificate
    client_key             = kind_cluster.this.client_key
    cluster_ca_certificate = kind_cluster.this.cluster_ca_certificate
  }
  git = {
    url = "ssh://git@github.com/${var.github_owner}/${var.flux_github_repo}.git"
    ssh = {
      username    = "git"
      private_key = tls_private_key.ecdsa-p384-key.private_key_pem
    }
  }
}

resource "kind_cluster" "this" {
  name           = "flux0"
  wait_for_ready = true
}

resource "tls_private_key" "ecdsa-p384-key" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P384"
}

resource "github_repository_deploy_key" "this" {
  title      = "Repository flux key"
  repository = var.flux_github_repo
  key        = tls_private_key.ecdsa-p384-key.public_key_openssh
  read_only  = "false"
}

resource "flux_bootstrap_git" "this" {
  depends_on = [github_repository_deploy_key.this]

  path = "clusters/kbot"

  components_extra = ["image-reflector-controller", "image-automation-controller"]
}
