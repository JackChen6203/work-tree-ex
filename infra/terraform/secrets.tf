# ── Database Password ────────────────────────────────────────
resource "google_secret_manager_secret" "db_password" {
  secret_id = "travel-planner-db-password-${var.environment}"

  replication {
    auto {}
  }

  labels = {
    environment = var.environment
    service     = "travel-planner"
  }
}

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = random_password.db_password.result
}

# ── JWT Secret ───────────────────────────────────────────────
resource "google_secret_manager_secret" "jwt_secret" {
  secret_id = "travel-planner-jwt-secret-${var.environment}"

  replication {
    auto {}
  }

  labels = {
    environment = var.environment
    service     = "travel-planner"
  }
}

resource "google_secret_manager_secret_version" "jwt_secret" {
  secret      = google_secret_manager_secret.jwt_secret.id
  secret_data = random_password.jwt_secret.result
}

resource "random_password" "jwt_secret" {
  length  = 64
  special = false
}

# ── LLM Encryption Key ──────────────────────────────────────
resource "google_secret_manager_secret" "llm_encryption_key" {
  secret_id = "travel-planner-llm-enc-key-${var.environment}"

  replication {
    auto {}
  }

  labels = {
    environment = var.environment
    service     = "travel-planner"
  }
}

resource "google_secret_manager_secret_version" "llm_encryption_key" {
  secret      = google_secret_manager_secret.llm_encryption_key.id
  secret_data = random_password.llm_encryption_key.result
}

resource "random_password" "llm_encryption_key" {
  length  = 32
  special = false
}
