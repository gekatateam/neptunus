package starlarkfs

/*Package starlarkfs implements os.ReadFile and os.ReadDir functions in Starlark

- `read_file(filename String) String` returns specified file content as string
- `read_dir(path String) List[String]` returns list of dir entries

*/

import (
	"os"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

var Module = &starlarkstruct.Module{
	Name: "fs",
	Members: starlark.StringDict{
		"read_file": starlark.NewBuiltin("read_file", ReadFile),
		"read_dir":  starlark.NewBuiltin("read_dir", ReadDir),
	},
}

func ReadFile(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var filename string
	if err := starlark.UnpackPositionalArgs("read_file", args, kwargs, 1, &filename); err != nil {
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
	if err := starlark.UnpackPositionalArgs("read_dir", args, kwargs, 1, &path); err != nil {
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
