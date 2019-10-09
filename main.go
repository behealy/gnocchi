package main

import (
	"bufio"
	"crypto/sha512"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/behealy/gnocchi/program"
	"github.com/howeyc/gopass"
)

func main() {
	debug := *flag.Bool("debug", false, "Boolean - Set debug to view verbose logs.")
	flag.Parse()

	prog, err := program.StartProgram(debug)
	if err != nil {
		fmt.Println("NO! It didn't WORK!")
		fmt.Println(err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Welcome, to PORDS")
	if prog.Debug == true {
		fmt.Println("DEBUG MODE ON")
	}
	fmt.Println("---------------------")

	for {
		if prog.MasterPwIsSet() != true {
			fmt.Print("Please enter master password: ")
			text, err := gopass.GetPasswdMasked()
			if err != nil {
				// Handle gopass.ErrInterrupted or getch() read error
			}
			if len(text) > 0 {
				hpw := sha512.Sum512_256(text)
				if err == nil {
					prog.MasterPw = hpw[:]
				} else {
					fmt.Println(err)
				}
			}

		} else {
			prog.PrintPrompt()
			text, _ := reader.ReadString('\n')
			// convert CRLF to LF
			if runtime.GOOS == "windows" {
				text = strings.TrimRight(text, "\r\n")
			} else {
				text = strings.TrimRight(text, "\n")
			}
			text = strings.Replace(text, "\n", "", -1)

			err := prog.HandleInput(text)
			if err != nil {
				fmt.Println(err.Error())
				break
			}
		}
	}
}
