package main

import (
	"fmt"
	"github.com/gofunct/goexec"
	"os"
)

func init() {

}

var (
	exe = goexec.NewCommand("cloudscript", "a scripting utility tool for cloud development", "0.1")
)

func main() {
	exe.Act("setup", "set gcloud project", func(cmd *goexec.Command) error {
		_ = cmd.Prompt("project", "what is the id of your gcloud project?")
		cmd.AddScript(`gcloud config set project {{ .project }}`)
		cmd.AddScript(`gcloud components update`)
		return cmd.Run()
	})
	exe.Act("install", "install cloudctl tool", func(cmd *goexec.Command) error {
		cmd.AddScript(`go mod vendor`)
		cmd.AddScript(`go generate {{ .pwd }}/...`)
		cmd.AddScript(`go fmt {{ .pwd }}/...`)
		cmd.AddScript(`go install {{ .gopath }}/src/{{ .target }}`)
		return cmd.Run()
	})
	exe.Act("rebuild", "rebuild this cloudscript tool", func(cmd *goexec.Command) error {
		cmd.AddScript(`go mod vendor`)
		cmd.AddScript(`go fmt {{ .pwd }}/...`)
		cmd.AddScript(`go install {{ .gopath }}/src/{{ .rebuild }}`)
		return cmd.Run()
	})
	exe.Act("token", "rebuild this cloudscript tool", func(cmd *goexec.Command) error {
		cmd.AddScript(`gcloud auth application-default print-access-token
`)
		return cmd.Run()
	})

	if err := exe.Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
