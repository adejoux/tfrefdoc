# TODO(tfrefdoc): generated skeleton: replace the placeholder values and remove this line.
resource "acme_server" "example" {
  name       = "example"
  network_id = acme_network.example.id
  disks = [
    {
      size_gb = 1
      # encrypted = true
    },
  ]

  rule {
    port = 443
    # protocol = "tcp"

    # source {
    #   cidr = "10.0.0.0/16"
    # }
  }

  # Optional arguments:
  # password = var.password
  # size     = "small"
  # tags     = { environment = "dev" }
  # settings = {
  #   backup = {
  #     retention_days = 30
  #     schedule = {
  #       cron     = "example"
  #       timezone = "example"
  #     }
  #   }
  # }

  # label "key" {
  #   value = "example"
  # }

  # timeouts {
  #   create = "30m"
  #   delete = "example"
  # }
}
