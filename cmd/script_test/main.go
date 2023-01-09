package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/flexlet/utils"
)

func main() {
	var (
		scriptPath string
		workDir    string
		scriptArgs string
		scriptEnv  string
		scriptOut  string
	)
	flag.StringVar(&scriptPath, "path", "/home/yao/test.sh", "Script path")
	flag.StringVar(&workDir, "dir", "/home/yao/", "Script work dir")
	flag.StringVar(&scriptArgs, "args", "-m hello", "Script arguments")
	flag.StringVar(&scriptEnv, "env", "EXIT_CODE_OK=0 ERROR_CODE_INVALID=1", "Script enviorment variables")
	flag.StringVar(&scriptOut, "out", "/home/yao/test.out", "Script output")
	flag.Parse()
	TestScript(scriptPath, workDir, scriptArgs, scriptEnv, scriptOut)
}

func TestScript(scriptPath string, workDir string, scriptArgs string, scriptEnv string, scriptOut string) {
	scriptSpec := utils.ScriptSpec{
		Path: scriptPath,
		Dir:  workDir,
		Args: strings.Split(scriptArgs, " "),
		Env:  strings.Split(scriptEnv, " "),
		Out:  scriptOut,
	}

	var script *utils.Script
	var err error

	script, err = utils.NewScript(&scriptSpec, func() {
		fmt.Printf("script.Callback, exit code: %d\n", script.Cmd.ProcessState.ExitCode())
	})

	if err != nil {
		panic(err)
	}

	pool := utils.DefaultThreadPool()
	if err := pool.Put(script); err != nil {
		panic(err)
	}

	println("script.Wait")
	script.Wait()

	// sleep to wait callback messaage
	time.Sleep(time.Millisecond * 100)

	println("pool.Destroy")
	pool.Destroy()
}
