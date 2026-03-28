package quant

import (
	"context"
	_ "embed"
	"fmt"
	"path"

	pf "github.com/pulumi/pulumi-terraform-bridge/v3/pkg/pf/tfbridge"
	"github.com/pulumi/pulumi-terraform-bridge/v3/pkg/tfbridge"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"

	"github.com/quantcdn/pulumi-quant/provider/pkg/version"
	"github.com/quantcdn/terraform-provider-quant/v5/shim"
)

// all of the token components used below.
const (
	mainPkg = "quant"
	mainMod = "index"
)

//go:embed cmd/pulumi-resource-quant/bridge-metadata.json
var metadata []byte

// intIDComputeID converts an integer "id" field from Terraform state to a string
// resource ID for Pulumi. Required because Pulumi expects all resource IDs to be strings.
func intIDComputeID(_ context.Context, state resource.PropertyMap) (resource.ID, error) {
	idProp, ok := state["id"]
	if !ok {
		return "", fmt.Errorf("missing id property in state")
	}
	if idProp.IsNumber() {
		return resource.ID(fmt.Sprintf("%d", int64(idProp.NumberValue()))), nil
	}
	if idProp.IsString() {
		return resource.ID(idProp.StringValue()), nil
	}
	return "", fmt.Errorf("unexpected id type: %v", idProp.TypeString())
}

// fieldComputeID returns a ComputeID function that reads the resource ID from a
// named field in the TF state. Used for resources that don't have an "id" attribute.
func fieldComputeID(fieldName string) tfbridge.ComputeID {
	return func(_ context.Context, state resource.PropertyMap) (resource.ID, error) {
		prop, ok := state[resource.PropertyKey(fieldName)]
		if !ok {
			return "", fmt.Errorf("missing %s property in state", fieldName)
		}
		if prop.IsString() {
			return resource.ID(prop.StringValue()), nil
		}
		return "", fmt.Errorf("unexpected %s type: %v", fieldName, prop.TypeString())
	}
}

// Provider returns additional overlaid schema and metadata associated with the provider.
func Provider() tfbridge.ProviderInfo {
	prov := tfbridge.ProviderInfo{
		// Plugin-framework provider — use pf.ShimProvider instead of shimv2.NewProvider.
		P:                pf.ShimProvider(shim.NewProvider()),
		Name:             "quant",
		DisplayName:      "QuantCDN",
		Version:          version.Version,
		Description:      "A Pulumi package for managing QuantCDN resources.",
		Publisher:        "QuantCDN",
		PluginDownloadURL: "github://api.github.com/quantcdn/pulumi-quant",
		Keywords:         []string{"pulumi", "quant", "quantcdn", "cdn", "category/cloud"},
		License:          "Apache-2.0",
		Homepage:         "https://www.quantcdn.io",
		Repository:       "https://github.com/quantcdn/pulumi-quant",
		GitHubOrg:        "quantcdn",
		MetadataInfo:     tfbridge.NewProviderMetadata(metadata),

		Config: map[string]*tfbridge.SchemaInfo{},

		Resources: map[string]*tfbridge.ResourceInfo{
			// Resources with integer IDs need ComputeID to convert to string.
			"quant_project":          {Tok: "quant:index:Project", ComputeID: intIDComputeID},
			"quant_domain":           {Tok: "quant:index:Domain", ComputeID: intIDComputeID},
			"quant_crawler":          {Tok: "quant:index:Crawler", ComputeID: intIDComputeID},
			"quant_crawler_schedule": {Tok: "quant:index:CrawlerSchedule", ComputeID: intIDComputeID},

			// Rule resources use UUID as their primary identifier.
			"quant_rule_proxy":           {Tok: "quant:index:RuleProxy", ComputeID: fieldComputeID("uuid")},
			"quant_rule_redirect":        {Tok: "quant:index:RuleRedirect", ComputeID: fieldComputeID("uuid")},
			"quant_rule_custom_response": {Tok: "quant:index:RuleCustomResponse", ComputeID: fieldComputeID("uuid")},
			"quant_rule_content_filter":  {Tok: "quant:index:RuleContentFilter", ComputeID: fieldComputeID("uuid")},
			"quant_rule_function":        {Tok: "quant:index:RuleFunction", ComputeID: fieldComputeID("uuid")},
			"quant_rule_auth":            {Tok: "quant:index:RuleAuth", ComputeID: fieldComputeID("uuid")},
			"quant_rule_bot_challenge":   {Tok: "quant:index:RuleBotChallenge", ComputeID: fieldComputeID("uuid")},
			"quant_rule_headers":         {Tok: "quant:index:RuleHeaders", ComputeID: fieldComputeID("uuid")},
			"quant_rule_serve_static":    {Tok: "quant:index:RuleServeStatic", ComputeID: fieldComputeID("uuid")},

			"quant_header":      {Tok: "quant:index:Header"},
			"quant_application": {Tok: "quant:index:Application", ComputeID: fieldComputeID("appName")},
			"quant_environment": {Tok: "quant:index:Environment", ComputeID: fieldComputeID("envName")},
			"quant_volume":      {Tok: "quant:index:Volume", ComputeID: fieldComputeID("volumeId")},
			"quant_cron_job":    {Tok: "quant:index:CronJob", ComputeID: fieldComputeID("name")},
			"quant_kv_store":    {Tok: "quant:index:KvStore", ComputeID: fieldComputeID("storeId")},
			"quant_kv_item":     {Tok: "quant:index:KvItem", ComputeID: fieldComputeID("key")},
		},

		DataSources: map[string]*tfbridge.DataSourceInfo{
			"quant_project":  {Tok: "quant:index:getProject"},
			"quant_projects": {Tok: "quant:index:getProjects"},
		},

		JavaScript: &tfbridge.JavaScriptInfo{
			PackageName: "@quantcdn/pulumi-quant",
			Dependencies: map[string]string{
				"@pulumi/pulumi": "^3.0.0",
			},
			DevDependencies: map[string]string{
				"@types/node": "^10.0.0",
			},
			RespectSchemaVersion: true,
		},
		Python: &tfbridge.PythonInfo{
			PackageName:          "pulumi_quant",
			RespectSchemaVersion: true,
			PyProject:            struct{ Enabled bool }{true},
			Requires: map[string]string{
				"pulumi": ">=3.0.0,<4.0.0",
			},
		},
		Golang: &tfbridge.GolangInfo{
			ImportBasePath: path.Join(
				"github.com/quantcdn/pulumi-quant/sdk/",
				tfbridge.GetModuleMajorVersion(version.Version),
				"go",
				mainPkg,
			),
			GenerateResourceContainerTypes: true,
			GenerateExtraInputTypes:        true,
			RespectSchemaVersion:           true,
		},
		CSharp: &tfbridge.CSharpInfo{
			RespectSchemaVersion: true,
			PackageReferences: map[string]string{
				"Pulumi": "3.*",
			},
			RootNamespace: "QuantCDN",
			Namespaces: map[string]string{
				"quant": "Quant",
			},
		},
	}

	prov.MustApplyAutoAliases()
	prov.SetAutonaming(255, "-")

	return prov
}
