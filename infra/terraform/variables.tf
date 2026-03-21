# ── Project ──────────────────────────────────────────────────
variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "asia-east1"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

# ── Database ─────────────────────────────────────────────────
variable "db_tier" {
  description = "Cloud SQL instance tier"
  type        = string
  default     = "db-custom-2-8192"
}

variable "db_disk_size_gb" {
  description = "Cloud SQL disk size in GB"
  type        = number
  default     = 20
}

variable "db_name" {
  description = "Database name"
  type        = string
  default     = "travel_planner"
}

# ── Redis ────────────────────────────────────────────────────
variable "redis_memory_size_gb" {
  description = "Redis memory size in GB"
  type        = number
  default     = 1
}

variable "redis_tier" {
  description = "Redis tier (BASIC or STANDARD_HA)"
  type        = string
  default     = "BASIC"
}

# ── Cloud Run ────────────────────────────────────────────────
variable "api_image" {
  description = "Docker image for API service"
  type        = string
  default     = "gcr.io/PROJECT/travel-api:latest"
}

variable "worker_image" {
  description = "Docker image for Worker service"
  type        = string
  default     = "gcr.io/PROJECT/travel-worker:latest"
}

variable "api_min_instances" {
  description = "Minimum API instances"
  type        = number
  default     = 0
}

variable "api_max_instances" {
  description = "Maximum API instances"
  type        = number
  default     = 10
}
