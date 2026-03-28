package main

import (
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfgen"

	quant "github.com/quantcdn/pulumi-quant/provider"
)

func main() {
	tfgen.Main("quant", quant.Provider())
}
