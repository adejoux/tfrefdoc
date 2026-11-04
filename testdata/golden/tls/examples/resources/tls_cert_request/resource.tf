# TODO(tfrefdoc): generated skeleton: replace the placeholder values and remove this line.
resource "tls_cert_request" "example" {
  private_key_pem = tls_private_key.example.private_key_pem

  # Optional arguments:
  # dns_names                  = ["example.com"]
  # ip_addresses               = ["192.0.2.10"]
  # private_key_pem_wo         = var.private_key_pem_wo
  # private_key_pem_wo_version = 1
  # uris                       = ["https://example.com"]

  # subject {
  #   common_name         = "example"
  #   country             = "example"
  #   email_address       = "user@example.com"
  #   locality            = "example"
  #   organization        = "example"
  #   organizational_unit = "example"
  #   postal_code         = "example"
  #   province            = "example"
  #   serial_number       = "example"
  #   street_address      = ["example"]
  # }
}
