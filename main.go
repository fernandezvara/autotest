package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/deckarep/gosx-notifier"
	"gopkg.in/fsnotify.v1"
)

var (
	log *logrus.Logger
)

func main() {
	app := cli.NewApp()
	app.Name = "autotest"
	app.Usage = "run `go test` on file changes"
	app.Version = ""
	app.Author = "Antonio Fdez."
	app.Email = "github.com/fernandezvara"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "cmd",
			Value: "go test",
			Usage: "Command to exec on every change",
		},
		cli.StringFlag{
			Name:  "flags",
			Value: "",
			Usage: "Command Flags to add.",
		},
		cli.StringFlag{
			Name:  "path",
			Value: ".",
			Usage: "Path to watch for changes.",
		},
		cli.BoolFlag{
			Name:  "skip-notify",
			Usage: "Skip sending desktop notifications?",
		},
	}

	app.Before = func(c *cli.Context) error {
		log = logrus.New()
		log.Out = os.Stderr
		return nil
	}
	app.Action = startCmd
	app.Run(os.Args)
}

func startCmd(c *cli.Context) {
	cmd, flags := getCmdFlags(c)

	fmt.Println(cmd, flags)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				// log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					runTest(c, cmd, flags)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(".")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func runTest(c *cli.Context, cmd string, flags []string) {
	command := exec.Command(cmd, flags...)
	var stdout bytes.Buffer
	command.Stdout = &stdout
	err := command.Run()
	if err != nil {
		log.Errorln(err)
		if c.GlobalBool("skip-notify") == false {
			showNotification(true, err.Error())
		}
	}

	if c.GlobalBool("skip-notify") == false {
		showNotification(false, stdout.String())
	}
}

func showNotification(isErr bool, message string) {

	var (
		title string
		msg   string
	)

	if isErr == true {
		title = "ERROR"
		msg = message
	} else {
		lines := strings.Split(message, "\n")
		fmt.Println("len:", len(lines))
		for _, line := range lines {
			if line == "PASS" || line == "FAIL" {
				title = line
			}
		}
		msg = fmt.Sprintf("%s\n%s\n", lines[len(lines)-3], lines[len(lines)-2])
		// msg = strings.Join(lines[len(lines)-3:len(lines)-2], "\n")
		fmt.Println(msg)
	}

	note := gosxnotifier.NewNotification(msg)
	note.Title = title
	note.Sound = gosxnotifier.Default

	err := note.Push()
	if err != nil {
		panic(err)
	}
}

func getCmdFlags(c *cli.Context) (string, []string) {
	var (
		realCmd   string
		realFlags []string
	)

	if len(strings.Split(c.GlobalString("cmd"), " ")) > 1 {
		realCmd = strings.Split(c.GlobalString("cmd"), " ")[0]
		for id, val := range strings.Split(c.GlobalString("cmd"), " ") {
			if id != 0 {
				realFlags = append(realFlags, val)
			}
		}
		for _, val := range strings.Split(c.GlobalString("flags"), " ") {
			realFlags = append(realFlags, val)
		}
		return realCmd, realFlags
	}
	realCmd = c.GlobalString("cmd")
	realFlags = strings.Split(c.GlobalString("flags"), " ")
	return realCmd, realFlags
}
