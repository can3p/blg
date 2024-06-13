package main

import (
	"github.com/can3p/blg/cmd"

	_ "github.com/can3p/blg/pkg/services/livejournal"
	_ "github.com/can3p/blg/pkg/services/pcom"
)

func main() {
	cmd.Execute()
}
