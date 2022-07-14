source = ["releases/mac_darwin_all/acorn"]
bundle_id = "io.acorn.cli"

sign {
  application_identity = "Developer ID Application: Acorn Labs, Inc. (K5HKMU4T9S)"
}

zip {
  output_path = "releases/mac_darwin_all/acorn.zip"
}
