package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/format"
	"github.com/spf13/cobra"
)

type formatOptions struct {
	format    string
	noHeader  bool
	output    string
	separator string
}

func (options *formatOptions) register(command *cobra.Command) {
	command.Flags().StringVar(&options.format, "format", string(format.Text), "formato da saída: text, csv, tsv ou json")
	command.Flags().BoolVar(&options.noHeader, "no-header", false, "com csv ou tsv, omite a linha de nomes das colunas")
	command.Flags().StringVar(&options.output, "output", "", "caminho do arquivo a gravar, relativo ou absoluto, em vez do stdout; sem extensão, a do formato é acrescentada")
	command.Flags().StringVar(&options.separator, "separator", string(format.Comma), "separador do csv: , ; | ou \\t")
}

func (options formatOptions) resolve(command *cobra.Command) (rendering, error) {
	chosen, err := format.Parse(options.format)
	if err != nil {
		return rendering{}, err
	}

	if command.Flags().Changed("no-header") && !chosen.Delimited() {
		return rendering{}, fmt.Errorf("--no-header só vale com --format csv ou tsv, e veio --format %s", chosen)
	}

	separator, err := options.resolveSeparator(command, chosen)
	if err != nil {
		return rendering{}, err
	}

	return rendering{format: chosen, separator: separator, header: !options.noHeader}, nil
}

func (options formatOptions) resolveSeparator(command *cobra.Command, chosen format.Format) (format.Separator, error) {
	if !command.Flags().Changed("separator") {
		if chosen == format.TSV {
			return format.Tab, nil
		}
		return format.Comma, nil
	}

	if chosen != format.CSV {
		return 0, fmt.Errorf("--separator só vale com --format csv, e veio --format %s", chosen)
	}

	separator, err := format.ParseSeparator(options.separator)
	if err != nil {
		return 0, err
	}

	if separator == format.Tab {
		fmt.Fprintln(command.ErrOrStderr(), "aviso: csv com tabulação é o que --format tsv já faz")
	}

	return separator, nil
}
