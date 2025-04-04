package starlarkfs

import (
	"os"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var Module = &starlarkstruct.Module{
	Name: "fs",
	Members: starlark.StringDict{
		"readFile": starlark.NewBuiltin("readFile", ReadFile),
		"readDir":  starlark.NewBuiltin("readDir", ReadDir),
	},
}

func ReadFile(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var filename string
	if err := starlark.UnpackPositionalArgs("readFile", args, kwargs, 1, &filename); err != nil {
		return starlark.None, err
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return starlark.None, err
	}

	return starlark.String(content), nil
}

func ReadDir(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	if err := starlark.UnpackPositionalArgs("readDir", args, kwargs, 1, &path); err != nil {
		return starlark.None, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return starlark.None, err
	}

	entriesList := make([]starlark.Value, len(entries))
	for i, e := range entries {
		entriesList[i] = starlark.String(e.Name())
	}

	return starlark.NewList(entriesList), nil
}
