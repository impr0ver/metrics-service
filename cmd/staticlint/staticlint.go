// Main packet of staticlint utility, for static check golang code.
// staticlint use this public analyzers:
//   - bodyclose - check on close HTTP-response body;
//   - decorder - check style code declaration, order const, types, vars and functions;
//
// staticlint use this integrate analyzers:
//   - analysis/passes/printf - check format types in Printf parameters;
//   - analysis/passes/structtag - check struct tags;
//   - analysis/passes/shadow - check shadow variables;
//   - exitcheck - check on direct call os.Exit function in main of packet main.
//
// How to use:
// $ go vet -vettool=./cmd/staticlint/staticlint ./...
//
// For my analyzer test
// $ go vet -vettool=./cmd/staticlint/staticlint ./internal/pkg/exitcheck/testdata/pkg1/
package main

import (
	"strings"

	"github.com/impr0ver/metrics-service/internal/pkg/exitcheck" // packet of my analyzer realisation
	"github.com/timakin/bodyclose/passes/bodyclose"              // add one public analyzer, checks HTTP response body is closed successfully.
	"gitlab.com/bosi/decorder"                                   // add one public analyzer, a declaration order linter for golang. Declarations are type, const, var and func.
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	myStaticChecks := []*analysis.Analyzer{
		printf.Analyzer,
		//shadow.Analyzer, // this module generates too many false positives and is not yet enabled by default
		structtag.Analyzer,
		bodyclose.Analyzer,
		decorder.Analyzer,
		exitcheck.Analyzer, // add my analyzer (check on "os.Exit()") from "github.com/impr0ver/metrics-service/internal/pkg/exitcheck"
	}

	// add alalyzers from staticcheck packet in myStaticChecks.
	for _, v := range staticcheck.Analyzers {
		if strings.Contains(v.Analyzer.Name, "SA") || strings.Contains(v.Analyzer.Name, "ST") {
			myStaticChecks = append(myStaticChecks, v.Analyzer)
		}
	}
	multichecker.Main(
		myStaticChecks...,
	)
}
