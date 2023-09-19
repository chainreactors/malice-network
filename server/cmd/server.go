package cmd

import (
	"fmt"
	"github.com/jessevdk/go-flags"
)

func Run() {
	var opt ServerOptions
	parser := flags.NewParser(&opt, flags.Default)
	parser.Usage = Banner()
	_, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Println(err.Error())
		}
		return
	}
}

func Banner() string {
	return ""
}
