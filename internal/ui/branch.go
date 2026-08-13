package ui

import "github.com/LHPalma/gitarias/internal/branch"

func DescribeMerge(kind branch.MergeKind) string {
	switch kind {
	case branch.MergedBySquash:
		return "squashada"
	case branch.MergedByRebase:
		return "rebaseada"
	default:
		return "mergeada"
	}
}

func DescribeSource(source branch.BaseSource) string {
	switch source {
	case branch.BaseFromFlag:
		return "informada via --base"
	case branch.BaseFromOriginHead:
		return "detectada via origin/HEAD"
	default:
		return "encontrada localmente"
	}
}

func DescribeLayer(layer branch.Layer) string {
	if !layer.Merged {
		return "não mergeada"
	}

	return DescribeMerge(layer.Branch.Merge)
}
