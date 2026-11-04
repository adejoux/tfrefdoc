# TODO(tfrefdoc): generated skeleton: replace the placeholder values and remove this line.
resource "tls_self_signed_cert" "example" {
  allowed_uses          = ["any_extended"]
  private_key_pem       = tls_private_key.example.private_key_pem
  validity_period_hours = 24

  # Optional arguments:
  # dns_names                  = ["example.com"]
  # early_renewal_hours        = 0
  # ip_addresses               = ["192.0.2.10"]
  # is_ca_certificate          = false
  # max_path_length            = 1
  # private_key_pem_wo         = var.private_key_pem_wo
  # private_key_pem_wo_version = 1
  # set_authority_key_id       = false
  # set_subject_key_id         = false
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
