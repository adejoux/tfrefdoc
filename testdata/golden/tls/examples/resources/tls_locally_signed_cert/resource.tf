# TODO(tfrefdoc): generated skeleton: replace the placeholder values and remove this line.
resource "tls_locally_signed_cert" "example" {
  allowed_uses          = ["any_extended"]
  ca_cert_pem           = file("ca_cert.pem")
  ca_private_key_pem    = file("ca_private_key.pem")
  cert_request_pem      = tls_cert_request.example.cert_request_pem
  validity_period_hours = 24

  # Optional arguments:
  # ca_private_key_pem_wo         = var.ca_private_key_pem_wo
  # ca_private_key_pem_wo_version = 1
  # early_renewal_hours           = 0
  # is_ca_certificate             = false
  # max_path_length               = 1
  # set_subject_key_id            = false
}
