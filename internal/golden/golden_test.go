package golden

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielporterda/mintlify-fast-preview/internal/config"
	"github.com/danielporterda/mintlify-fast-preview/internal/preview"
	"github.com/danielporterda/mintlify-fast-preview/internal/site"
)

type goldenCase struct {
	name  string
	route string
}

func TestMintlifyGoldenComparisons(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "docs-main")
	cases := []goldenCase{
		{name: "index", route: "/"},
		{name: "api-reference", route: "/api-reference"},
		{name: "protobuf-operation", route: "/reference/protobuf/operations/com-digitalasset-canton-admin-mediator-v30/mediatorstatusservice/mediatorstatus"},
		{name: "openrpc-operation", route: "/reference/wallet-gateway-json-rpc/operations/dapp-api/connect"},
		{name: "asyncapi-operation", route: "/reference/json-api-asyncapi-reference/operations/v2-commands-completions/subscribe"},
		{name: "snippet-heavy-troubleshooting", route: "/global-synchronizer/troubleshooting-guide/common-questions"},
		{name: "custom-react-dashboard", route: "/shared/version-compatibility-dashboard"},
	}

	docs, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	routes, err := site.BuildRoutes(root, docs)
	if err != nil {
		t.Fatal(err)
	}
	byRoute := map[string]site.Route{}
	for _, route := range routes {
		byRoute[route.URL] = route
	}

	var failures []string
	for _, tc := range cases {
		route, ok := byRoute[tc.route]
		if !ok {
			failures = append(failures, tc.name+": missing route "+tc.route)
			continue
		}
		actual, err := preview.RenderPage(root, docs, route)
		if err != nil {
			failures = append(failures, tc.name+": render error: "+err.Error())
			continue
		}
		expectedPath := filepath.Join("..", "..", "testdata", "golden", tc.name+".html")
		expectedBytes, err := os.ReadFile(expectedPath)
		if err != nil {
			failures = append(failures, tc.name+": read expected: "+err.Error())
			continue
		}
		if normalize(actual) != normalize(string(expectedBytes)) {
			failures = append(failures, tc.name+": output differs from Mintlify baseline")
		}
	}
	if len(failures) > 0 {
		t.Fatalf("%d/%d golden tests failed\n%s", len(failures), len(cases), strings.Join(failures, "\n"))
	}
}

func normalize(input string) string {
	lines := strings.Fields(input)
	return strings.Join(lines, " ")
}
