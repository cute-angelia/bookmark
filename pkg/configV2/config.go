package configV2

import (
	"bookmark/pkg/env"
	_ "embed"
	"github.com/cute-angelia/go-utils/utils/conf"
)

//go:embed config.product.toml
var configProduct []byte

//go:embed config.local.toml
var configLocal []byte

func InitConfig(envStr string) {
	if env.IsLocal() || envStr == "local" {
		conf.MustLoadConfigByte(configLocal, "toml")
	} else {
		conf.MustLoadConfigByte(configProduct, "toml")
	}
}
