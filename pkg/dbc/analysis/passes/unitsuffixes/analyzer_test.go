package unitsuffixes

import (
	"testing"
	"text/scanner"

	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, Analyzer(), []*analysistest.Case{
		{
			Name: "ok",
			Data: `
BO_ 400 TestMessage: 3 ECU1
 SG_ ValuePercent : 0|1@1+ (1,0) [0|0] "%" DRIVER,IO
`,
		},

		{
			Name: "not ok",
			Data: `
BO_ 400 TestMessage: 3 ECU1
 SG_ ValuePct : 0|1@1+ (1,0) [0|0] "%" DRIVER,IO
`,
			Diagnostics: []*analysis.Diagnostic{
				{
					Pos:     scanner.Position{Line: 2, Column: 2},
					Message: "signal with unit % must have suffix Percent",
				},
			},
		},
	})
}
