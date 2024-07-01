package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/scanner"

	"github.com/alecthomas/kingpin/v2"
	"github.com/fatih/color"
	"github.com/xiaoxu5271/can-go/dbc/analysis/passes/uniquemessageids"
	"github.com/xiaoxu5271/can-go/internal/generate"
	"github.com/xiaoxu5271/can-go/pkg/dbc"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/definitiontypeorder"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/intervals"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/lineendings"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/messagenames"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/multiplexedsignals"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/newsymbols"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/nodereferences"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/noreservedsignals"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/requireddefinitions"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/signalbounds"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/signalnames"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/singletondefinitions"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/siunits"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/uniquenodenames"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/uniquesignalnames"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/unitsuffixes"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/valuedescriptions"
	"github.com/xiaoxu5271/can-go/pkg/dbc/analysis/passes/version"
)

func main() {
	app := kingpin.New("cantool", "CAN tool for Go programmers")
	generateCommand(app)
	lintCommand(app)
	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func generateCommand(app *kingpin.Application) {
	command := app.Command("generate", "generate CAN messages")
	inputDir := command.
		Arg("input-dir", "input directory").
		Required().
		ExistingDir()
	outputDir := command.
		Arg("output-dir", "output directory").
		Required().
		String()
	command.Action(func(_ *kingpin.ParseContext) error {
		return filepath.Walk(*inputDir, func(p string, i os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if i.IsDir() || filepath.Ext(p) != ".dbc" {
				return nil
			}
			relPath, err := filepath.Rel(*inputDir, p)
			if err != nil {
				return err
			}
			outputFile := relPath + ".go"
			outputPath := filepath.Join(*outputDir, outputFile)
			return genGo(p, outputPath)
		})
	})
}

func lintCommand(app *kingpin.Application) {
	command := app.Command("lint", "lint DBC files")
	fileOrDir := command.
		Arg("file-or-dir", "DBC file or directory").
		Required().
		ExistingFileOrDir()
	command.Action(func(_ *kingpin.ParseContext) error {
		filesToLint, err := resolveFileOrDirectory(*fileOrDir)
		if err != nil {
			return err
		}
		var hasFailed bool
		for _, lintFile := range filesToLint {
			f, err := os.Open(lintFile)
			if err != nil {
				return err
			}
			source, err := io.ReadAll(f)
			if err != nil {
				return err
			}
			p := dbc.NewParser(f.Name(), source)
			if err := p.Parse(); err != nil {
				printError(source, err.Position(), err.Reason(), "parse")
				continue
			}
			for _, a := range analyzers() {
				pass := &analysis.Pass{
					Analyzer: a,
					File:     p.File(),
				}
				if err := a.Run(pass); err != nil {
					return err
				}
				hasFailed = hasFailed || len(pass.Diagnostics) > 0
				for _, d := range pass.Diagnostics {
					printError(source, d.Pos, d.Message, a.Name)
				}
			}
		}
		if hasFailed {
			return errors.New("one or more lint errors")
		}
		return nil
	})
}

func analyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		// TODO: Re-evaluate if we want boolprefix.Analyzer(), since it creates a lot of churn in vendor schemas
		definitiontypeorder.Analyzer(),
		intervals.Analyzer(),
		lineendings.Analyzer(),
		messagenames.Analyzer(),
		multiplexedsignals.Analyzer(),
		newsymbols.Analyzer(),
		nodereferences.Analyzer(),
		noreservedsignals.Analyzer(),
		requireddefinitions.Analyzer(),
		signalbounds.Analyzer(),
		signalnames.Analyzer(),
		singletondefinitions.Analyzer(),
		siunits.Analyzer(),
		uniquemessageids.Analyzer(),
		uniquenodenames.Analyzer(),
		uniquesignalnames.Analyzer(),
		unitsuffixes.Analyzer(),
		valuedescriptions.Analyzer(),
		version.Analyzer(),
	}
}

func genGo(inputFile, outputFile string) error {
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return err
	}
	input, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}
	result, err := generate.Compile(inputFile, input)
	if err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		return warning
	}
	output, err := generate.Database(result.Database)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputFile, output, 0o600); err != nil {
		return err
	}
	fmt.Println("wrote:", outputFile)
	return nil
}

func resolveFileOrDirectory(fileOrDirectory string) ([]string, error) {
	fileInfo, err := os.Stat(fileOrDirectory)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return []string{fileOrDirectory}, nil
	}
	var files []string
	if err := filepath.Walk(fileOrDirectory, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() && filepath.Ext(path) == ".dbc" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func printError(source []byte, pos scanner.Position, msg, name string) {
	fmt.Printf("\n%s: %s (%s)\n", pos, color.RedString("%s", msg), name)
	fmt.Printf("%s\n", getSourceLine(source, pos))
	fmt.Printf("%s\n", caretAtPosition(pos))
}

func getSourceLine(source []byte, pos scanner.Position) []byte {
	lineStart := pos.Offset
	for lineStart > 0 && source[lineStart-1] != '\n' {
		lineStart--
	}
	lineEnd := pos.Offset
	for lineEnd < len(source) && source[lineEnd] != '\n' {
		lineEnd++
	}
	return source[lineStart:lineEnd]
}

func caretAtPosition(pos scanner.Position) string {
	return strings.Repeat(" ", pos.Column-1) + color.YellowString("^")
}
