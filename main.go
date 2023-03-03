package main

import (
	"fmt"
	"os"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"strings"
)

func main() {
	var f *os.File
	var err error

	if len(os.Args) < 2 {
		fmt.Println("please supply filename(s)")
		os.Exit(1)
	}

	froms2dockerfile := make(map[string][]string)
	for _, fn := range os.Args[1:] {
		f, err = os.Open(fn)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		froms := parseDockerfile(f)
		for _, from := range froms {
			fns := append(froms2dockerfile[from], fn)
			froms2dockerfile[from] = fns
		}
	}

	checkInconsistencies(froms2dockerfile)
}

func checkInconsistencies(froms2dockerfile map[string][]string) {
	type tagFile struct {
		tag      string
		file     string
		fullname string
	}

	image2TagFile := make(map[string][]tagFile)
	for from, files := range froms2dockerfile {
		for _, file := range files {
			splits := strings.Split(from, ":")
			if len(splits) < 1 {
				continue
			}
			tag := splits[len(splits)-1]
			image := strings.Join(splits[:len(splits)-1], "")
			if image == "" {
				continue
			}
			tf := tagFile{
				tag:      tag,
				file:     file,
				fullname: from,
			}
			image2TagFile[image] = append(image2TagFile[image], tf)
		}
	}

	for _, tfs := range image2TagFile {
		for i := 1; i < len(tfs); i++ {
			tf := tfs[i]
			if tf.tag != tfs[i-1].tag {
				fmt.Printf("tags differ for image %s in files %s and %s\n", tf.fullname, tf.file, tfs[i-1].file)
				os.Exit(1)
			}
		}
	}
}

func parseDockerfile(f *os.File) []string {
	var froms []string
	result, err := parser.Parse(f)
	if err != nil {
		panic(err)
	}

	vars := parseVars(result)
	_ = vars
	var next *parser.Node
	for _, node := range result.AST.Children {
		if node.Value == "FROM" {
			next = node.Next
			next = node.Next
			if next == nil {
				break
			}
			from := expandVariables(next, vars)
			froms = append(froms, from)
		}
	}

	return froms
}

func expandVariables(next *parser.Node, vars map[string]string) string {
	from := next.Value
	for key, val := range vars {
		from = strings.ReplaceAll(from, fmt.Sprintf("${%s}", key), val)
	}
	return from
}

func parseVars(result *parser.Result) map[string]string {
	vars := make(map[string]string)
	_, metaArgs, err := instructions.Parse(result.AST)
	if err != nil {
		panic(err)
	}

	for _, argCmd := range metaArgs {
		if argCmd.Name() != "ARG" {
			continue
		}
		for _, argCmdArg := range argCmd.Args {

			vars[argCmdArg.Key] = argCmdArg.ValueString()
		}
	}

	return vars
}
