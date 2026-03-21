# ── API ──────────────────────────────────────────────────────
output "api_url" {
  description = "API service URL"
  value       = google_cloud_run_v2_service.api.uri
}

# ── Database ─────────────────────────────────────────────────
output "db_private_ip" {
  description = "Cloud SQL private IP"
  value       = google_sql_database_instance.main.private_ip_address
}

output "db_connection_name" {
  description = "Cloud SQL connection name"
  value       = google_sql_database_instance.main.connection_name
}

# ── Redis ────────────────────────────────────────────────────
output "redis_host" {
  description = "Redis instance host"
  value       = google_redis_instance.main.host
}

output "redis_port" {
  description = "Redis instance port"
  value       = google_redis_instance.main.port
}
