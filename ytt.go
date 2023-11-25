package main

import (
	"fmt"
	yttcmd "github.com/vmware-tanzu/carvel-ytt/pkg/cmd/template"
	yttui "github.com/vmware-tanzu/carvel-ytt/pkg/cmd/ui"
	yttfiles "github.com/vmware-tanzu/carvel-ytt/pkg/files"
)

func ytt(tpl []string) (string, error) {
	// create and invoke ytt "template" command
	templatingOptions := yttcmd.NewOptions()

	input, err := templatesAsInput(tpl...)
	if err != nil {
		return "", err
	}
	
	// for in-memory use, pipe output to "/dev/null"
	noopUI := yttui.NewCustomWriterTTY(false, noopWriter{}, noopWriter{})

	// Evaluate the template given the configured data values...
	output := templatingOptions.RunWithFiles(input, noopUI)
	if output.Err != nil {
		return "", output.Err
	}

	// output.DocSet contains the full set of resulting YAML documents, in order.
	bs, err := output.DocSet.AsBytes()
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

// templatesAsInput conveniently wraps one or more strings, each in a files.File, into a template.Input.
func templatesAsInput(tpl ...string) (yttcmd.Input, error) {
	var files []*yttfiles.File
	for i, t := range tpl {
		// to make this less brittle, you'll probably want to use well-defined names for `path`, here, for each input.
		// this matters when you're processing errors which report based on these paths.
		file, err := yttfiles.NewFileFromSource(yttfiles.NewBytesSource(fmt.Sprintf("tpl%d.yml", i), []byte(t)))
		if err != nil {
			return yttcmd.Input{}, err
		}

		files = append(files, file)
	}

	return yttcmd.Input{Files: files}, nil
}

type noopWriter struct{}

func (w noopWriter) Write(data []byte) (int, error) { return len(data), nil }
