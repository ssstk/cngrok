package client

import (
	"cngrok/log"
	"cngrok/util"
	"fmt"
	"github.com/inconshreveable/mousetrap"
	"math/rand"
	"os"
	"runtime"
	"time"
)

func init() {
	if runtime.GOOS == "windows" {
		if mousetrap.StartedByExplorer() {
			fmt.Println("请不要直接双击运行ngrok!")
			fmt.Println("你需要打开cmd.exe并从命令行中运行!")
			time.Sleep(5 * time.Second)
			os.Exit(1)

		}

	}
}
func Main() {
	// parse options
	opts, err := ParseArgs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// set up logging
	log.LogTo(opts.logto, opts.loglevel)

	// read configuration file
	config, err := LoadConfiguration(opts)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// seed random number generator
	seed, err := util.RandomSeed()
	if err != nil {
		fmt.Printf("无法安全地生成随机数生成器！")
		os.Exit(1)
	}
	rand.Seed(seed)

	NewController().Run(config)
}
