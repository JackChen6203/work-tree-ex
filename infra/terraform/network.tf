# ── VPC ──────────────────────────────────────────────────────
resource "google_compute_network" "main" {
  name                    = "travel-planner-vpc-${var.environment}"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "main" {
  name          = "travel-planner-subnet-${var.environment}"
  ip_cidr_range = "10.0.0.0/20"
  region        = var.region
  network       = google_compute_network.main.id

  private_ip_google_access = true
}

# ── VPC Connector (for Cloud Run → VPC access) ──────────────
resource "google_vpc_access_connector" "main" {
  name          = "travel-vpc-conn-${var.environment}"
  region        = var.region
  network       = google_compute_network.main.name
  ip_cidr_range = "10.8.0.0/28"

  min_instances = 2
  max_instances = 3
}

# ── Private Service Access (for Cloud SQL) ──────────────────
resource "google_compute_global_address" "private_ip" {
  name          = "travel-private-ip-${var.environment}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.main.id
}

resource "google_service_networking_connection" "private" {
  network                 = google_compute_network.main.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip.name]
}
