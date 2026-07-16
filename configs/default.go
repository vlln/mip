package configs

import _ "embed"

//go:embed mip.yaml
var Official []byte

//go:embed gip.yaml
var OfficialGIP []byte
