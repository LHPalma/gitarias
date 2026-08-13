package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/LHPalma/gitarias/internal/branch"
	"github.com/LHPalma/gitarias/internal/ui"
)

type treeTable struct {
	base   branch.Base
	layers []branch.Layer
}

func (data treeTable) header() []string {
	return []string{"branch", "pai", "estado"}
}

func (data treeTable) rows() [][]string {
	rows := make([][]string, 0, len(data.layers))
	for _, layer := range data.ordered() {
		rows = append(rows, []string{layer.Branch.Name, data.parentName(layer), ui.DescribeLayer(layer)})
	}

	return rows
}

func (data treeTable) document() any {
	records := make([]layerRecord, 0, len(data.layers))
	for _, layer := range data.ordered() {
		records = append(records, layerRecord{
			Branch: layer.Branch.Name,
			Parent: data.parentName(layer),
			Merged: layer.Merged,
			Merge:  mergeToken(layer),
		})
	}

	return treeDocument{
		Base:   baseRecord{Name: data.base.Name, Source: data.base.Source.String()},
		Layers: records,
	}
}

func mergeToken(layer branch.Layer) string {
	if !layer.Merged {
		return ""
	}

	return layer.Branch.Merge.String()
}

func (data treeTable) parentName(layer branch.Layer) string {
	if layer.Parent == "" {
		return data.base.Name
	}

	return layer.Parent
}

func (data treeTable) text(output io.Writer) error {
	if len(data.layers) == 0 {
		_, err := fmt.Fprintln(output, "Nenhuma branch local além da base.")
		return err
	}

	if _, err := fmt.Fprintln(output, data.base.Name); err != nil {
		return err
	}

	writer := columns(output)
	data.branch(writer, "", "")

	return writer.Flush()
}

func (data treeTable) branch(output io.Writer, parent string, prefix string) {
	children := data.childrenOf(parent)

	for index, child := range children {
		connector, descent := "├─ ", "│  "
		if index == len(children)-1 {
			connector, descent = "└─ ", "   "
		}

		fmt.Fprintf(output, "%s%s%s\t%s\n", prefix, connector, child.Branch.Name, ui.DescribeLayer(child))
		data.branch(output, child.Branch.Name, prefix+descent)
	}
}

func (data treeTable) childrenOf(parent string) []branch.Layer {
	var children []branch.Layer
	for _, layer := range data.layers {
		if layer.Parent == parent {
			children = append(children, layer)
		}
	}

	sort.Slice(children, func(one int, other int) bool {
		return children[one].Branch.Name < children[other].Branch.Name
	})

	return children
}

func (data treeTable) ordered() []branch.Layer {
	var walk func(parent string) []branch.Layer
	walk = func(parent string) []branch.Layer {
		var found []branch.Layer
		for _, child := range data.childrenOf(parent) {
			found = append(found, child)
			found = append(found, walk(child.Branch.Name)...)
		}

		return found
	}

	ordered := walk("")
	if len(ordered) == len(data.layers) {
		return ordered
	}

	return data.layers
}
