package main

import (
	"fmt"
	"starevent/star"

	"go.starlark.net/starlark"
)

var src = `
def process():
	e = newEvent()

	print(type(e))
	print(e)
	e.setRK("new key")
	print(e.getRK())

`

func main() {
	builtins := starlark.StringDict{}
	builtins["newEvent"] = starlark.NewBuiltin("newEvent", star.NewEvent)
	thread := &starlark.Thread{
		Print: func(_ *starlark.Thread, msg string) {
			fmt.Printf("from starlark program: %v\n", msg)
		},
	}

	_, program, err := starlark.SourceProgram("test.star", src, builtins.Has)
	if err != nil {
		panic(err)
	}

	globals, err := program.Init(thread, builtins)
	if err != nil {
		panic(err)
	}
	
	_, err = starlark.Call(thread, globals["process"], nil, nil)
	if err != nil {
		panic(err)
	}
}
