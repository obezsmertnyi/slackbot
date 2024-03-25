variable "github_owner" {
  description = "The name GitHub owner"
  default     = "obezsmertnyi"
  type        = string
}

variable "flux_github_repo" {
  description = "The name flux repo"
  default     = "flux-gitops"
  type        = string
}

variable "github_token" {
  description = "The token for the GitHub"
  sensitive   = true
  type        = string
}

variable "config_path" {
  description = "The path to the kubeconfig file"
  type        = string
  default     = "~/.kube/config"
}