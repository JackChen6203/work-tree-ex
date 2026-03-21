# ── Memorystore Redis ────────────────────────────────────────
resource "google_redis_instance" "main" {
  name               = "travel-planner-redis-${var.environment}"
  region             = var.region
  tier               = var.redis_tier
  memory_size_gb     = var.redis_memory_size_gb
  redis_version      = "REDIS_7_2"
  authorized_network = google_compute_network.main.id

  redis_configs = {
    maxmemory-policy = "allkeys-lru"
  }

  maintenance_policy {
    weekly_maintenance_window {
      day = "SUNDAY"
      start_time {
        hours   = 4
        minutes = 0
      }
    }
  }

  labels = {
    environment = var.environment
    service     = "travel-planner"
  }
}
