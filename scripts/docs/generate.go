package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/segmentio/terraform-docs/cmd"
	"github.com/segmentio/terraform-docs/internal/format"
	"github.com/segmentio/terraform-docs/internal/module"
	"github.com/segmentio/terraform-docs/pkg/print"
	"github.com/spf13/cobra"
)

// These are practiaclly a copy/paste of https://github.com/spf13/cobra/blob/master/doc/md_docs.go
// The reason we've decided to bring them over and not use them directly from cobra module was
// that we wanted to inject custom "Example" section with generated output based on the "examples"
// folder.

func main() {
	for _, c := range cmd.FormatterCmds() {
		err := generate(c, "./docs/formats")
		if err != nil {
			log.Fatal(err)
		}
	}
}

func generate(cmd *cobra.Command, dir string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := generate(c, dir); err != nil {
			return err
		}
	}

	cmdpath := strings.Replace(cmd.CommandPath(), "terraform-docs ", "", -1)
	basename := strings.Replace(cmdpath, " ", "-", -1) + ".md"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	if _, err := io.WriteString(f, ""); err != nil {
		return err
	}
	if err := generateMarkdown(cmd, f); err != nil {
		return err
	}
	return nil
}

func generateMarkdown(cmd *cobra.Command, w io.Writer) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()

	short := cmd.Short
	long := cmd.Long
	if len(long) == 0 {
		long = short
	}

	buf.WriteString("## " + name + "\n\n")
	buf.WriteString(short + "\n\n")
	buf.WriteString("### Synopsis\n\n")
	buf.WriteString(long + "\n\n")

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("### Examples\n\n")
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.Example))
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}

	err := printExample(buf, name)
	if err != nil {
		return err
	}

	if !cmd.DisableAutoGenTag {
		buf.WriteString("###### Auto generated by spf13/cobra on " + time.Now().Format("2-Jan-2006") + "\n")
	}
	_, err = buf.WriteTo(w)
	return err
}

func printOptions(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("### Options\n\n```\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("### Options inherited from parent commands\n\n```\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n\n")
	}
	return nil
}

func getPrinter(name string, settings *print.Settings) print.Format {
	switch strings.Replace(name, "terraform-docs ", "", -1) {
	case "json":
		return format.NewJSON(settings)
	case "markdown document":
		return format.NewDocument(settings)
	case "markdown table":
		return format.NewTable(settings)
	case "pretty":
		return format.NewPretty(settings)
	case "xml":
		return format.NewXML(settings)
	case "yaml":
		return format.NewYAML(settings)
	}
	return nil
}

func getFlags(name string) string {
	switch strings.Replace(name, "terraform-docs ", "", -1) {
	case "pretty":
		return " --no-color"
	}
	return ""
}

func printExample(buf *bytes.Buffer, name string) error {
	buf.WriteString("### Example\n\n")
	buf.WriteString("Given the [`examples`](/examples/) module:\n\n")
	buf.WriteString("```shell\n")
	buf.WriteString(fmt.Sprintf("%s%s ./examples/\n", name, getFlags(name)))
	buf.WriteString("```\n\n")
	buf.WriteString("generates the following output:\n\n")

	settings := print.NewSettings()
	settings.ShowColor = false
	options := &module.Options{
		Path: "./examples",
		SortBy: &module.SortBy{
			Name:     settings.SortByName,
			Required: settings.SortByRequired,
		},
	}
	tfmodule, err := module.LoadWithOptions(options)
	if err != nil {
		log.Fatal(err)
	}

	if printer := getPrinter(name, settings); printer != nil {
		output, err := printer.Print(tfmodule, settings)
		if err != nil {
			return err
		}
		segments := strings.Split(output, "\n")
		for _, s := range segments {
			if s == "" {
				buf.WriteString("\n")
			} else {
				buf.WriteString(fmt.Sprintf("    %s\n", s))
			}
		}
	}

	buf.WriteString("\n\n")
	return nil
}
