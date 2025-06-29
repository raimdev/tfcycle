package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	version = "1.0.0"
	usage   = `tfcycle - Terraform Cycle Error Analyzer

USAGE:
    tfcycle [COMMAND] [OPTIONS]

COMMANDS:
    analyze     Analyze Terraform cycle error (default)
    visualize   Generate DOT visualization of cycle
    version     Show version information
    help        Show this help message

OPTIONS:
    --error-file FILE    Read error from file instead of stdin
    --output FILE        Write output to file instead of stdout
    --verbose           Show detailed analysis
    --json              Output as JSON
    --help              Show help for command

EXAMPLES:
    # Analyze error from terraform output
    terraform plan 2>&1 | tfcycle analyze
    
    # Analyze error from file
    tfcycle analyze --error-file cycle_error.txt
    
    # Generate DOT visualization
    tfcycle visualize --output cycle.dot
    
    # Verbose JSON output
    tfcycle analyze --verbose --json

DESCRIPTION:
    tfcycle parses Terraform cycle error messages and provides clear, 
    actionable analysis of dependency cycles in Infrastructure as Code.
    It identifies minimal cycles and suggests common solutions.
`
)

type Config struct {
	Command   string
	ErrorFile string
	Output    string
	Verbose   bool
	JSON      bool
	Help      bool
}

func main() {
	config := parseArgs()
	
	if config.Help {
		fmt.Print(usage)
		return
	}
	
	if config.Command == "version" {
		fmt.Printf("tfcycle version %s\n", version)
		return
	}
	
	if config.Command == "help" {
		fmt.Print(usage)
		return
	}
	
	if err := runCommand(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs() Config {
	config := Config{
		Command: "analyze",
	}
	
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		config.Command = os.Args[1]
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	
	flag.StringVar(&config.ErrorFile, "error-file", "", "Read error from file instead of stdin")
	flag.StringVar(&config.Output, "output", "", "Write output to file instead of stdout")
	flag.BoolVar(&config.Verbose, "verbose", false, "Show detailed analysis")
	flag.BoolVar(&config.JSON, "json", false, "Output as JSON")
	flag.BoolVar(&config.Help, "help", false, "Show help")
	
	flag.Usage = func() {
		fmt.Print(usage)
	}
	
	flag.Parse()
	
	return config
}

func runCommand(config Config) error {
	switch config.Command {
	case "analyze":
		return runAnalyze(config)
	case "visualize":
		return runVisualize(config)
	default:
		return fmt.Errorf("unknown command: %s", config.Command)
	}
}

func runAnalyze(config Config) error {
	errorText, err := readInput(config.ErrorFile)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	parser := NewParser()
	cycle, err := parser.ParseError(errorText)
	if err != nil {
		return fmt.Errorf("failed to parse cycle error: %w", err)
	}
	
	analyzer := NewCycleAnalyzer(cycle)
	formatter := NewOutputFormatter(analyzer, config.Verbose)
	
	var output string
	if config.JSON {
		output, err = formatter.FormatAsJSON()
		if err != nil {
			return fmt.Errorf("failed to format as JSON: %w", err)
		}
	} else {
		output = formatter.FormatAnalysis()
	}
	
	return writeOutput(output, config.Output)
}

func runVisualize(config Config) error {
	errorText, err := readInput(config.ErrorFile)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	
	parser := NewParser()
	cycle, err := parser.ParseError(errorText)
	if err != nil {
		return fmt.Errorf("failed to parse cycle error: %w", err)
	}
	
	analyzer := NewCycleAnalyzer(cycle)
	formatter := NewOutputFormatter(analyzer, false)
	
	dotOutput := formatter.GenerateVisualization()
	if dotOutput == "" {
		return fmt.Errorf("no cycles found to visualize")
	}
	
	return writeOutput(dotOutput, config.Output)
}

func readInput(filename string) (string, error) {
	var reader io.Reader
	
	if filename != "" {
		file, err := os.Open(filename)
		if err != nil {
			return "", fmt.Errorf("failed to open file %s: %w", filename, err)
		}
		defer file.Close()
		reader = file
	} else {
		stat, err := os.Stdin.Stat()
		if err != nil {
			return "", fmt.Errorf("failed to stat stdin: %w", err)
		}
		
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return "", fmt.Errorf("no input provided. Use --error-file or pipe input to stdin")
		}
		
		reader = os.Stdin
	}
	
	var content strings.Builder
	scanner := bufio.NewScanner(reader)
	
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}
	
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	
	text := content.String()
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("input is empty")
	}
	
	return text, nil
}

func writeOutput(content, filename string) error {
	var writer io.Writer
	
	if filename != "" {
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filename, err)
		}
		defer file.Close()
		writer = file
	} else {
		writer = os.Stdout
	}
	
	_, err := writer.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	
	return nil
}