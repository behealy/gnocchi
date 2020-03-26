package program

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/mitchellh/go-homedir"
)

const (
	stateStartup ProgramState = iota
	stateMain
	stateCreateNew
	stateEdit
)

const (
	StartOfFile   byte = 0x2
	EndOfFile     byte = 0x3
	StartOfRecord byte = 0x1e
	StartOfEntry  byte = 0x1f
)

type parser struct {
	recordCursor int
	entryCursor  int
}

type GnocchiProgram struct {
	state         ProgramState
	MasterPw      []byte
	logins        []*LoginInfo
	loginsLut     map[string]*LoginInfo
	cacheFilePath string

	Debug bool

	newLoginChildProgram *NewLoginProgramModule
}

func help() {
	fmt.Println(`PORDS accepts the following commands:
	'new' - add a new account for which to save a password.
	'list' - list all accounts that you currently have saved for which passwords are generated.
	'get' - get password for an account. Enter the Site name value.
	'regen' - generate a new password for a given account.
	'edit' - change special characters used in password or the length of the password. (CAUTION: this will regenerate the password!s)
	`)
}

func (prog *GnocchiProgram) setState(st ProgramState) {
	prog.state = st
}

func (prog *GnocchiProgram) PrintPrompt() {
	switch prog.state {
	case stateMain:
		fmt.Print("PORDS-> ")
	case stateEdit:
		fallthrough
	case stateCreateNew:
		prog.newLoginChildProgram.PrintPrompt()
	}
}

func (prog *GnocchiProgram) HandleInput(input string) error {
	switch prog.state {
	case stateMain:
		return prog.handleInput(input)
	case stateCreateNew:
		fallthrough
	case stateEdit:
		err := prog.newLoginChildProgram.HandleInput(input)
		if err != nil {
			if _, ok := err.(ProgramExitError); ok == true {
				if err.Error() == "finished" {
					prog.addLoginInfo(prog.newLoginChildProgram.writtenLinfo)
					prog.write()
				}
			} else {
				fmt.Println(err.Error())
			}
			prog.state = stateMain
		}
	default:
		return prog.handleInput(input)
	}
	return nil
}

func (prog *GnocchiProgram) handleInput(input string) error {

	args := strings.Fields(strings.TrimSpace(input))
	in := args[0]

	switch in {
	case "help":
		help()
		return nil
	case "new":
		prog.setState(stateCreateNew)
		linfo := NewBlankLoginInfo()
		prog.newLoginChildProgram.NewLogin(&linfo)
		return nil
	case "get":
		if len(args) == 2 {
			if prog.loginsLut[args[1]] != nil {
				pw := prog.loginsLut[args[1]].GetAcctPassword(prog.MasterPw)
				clipboard.WriteAll(pw)
				fmt.Println("Password: " + pw + " copied to clipboard")
			} else {
				fmt.Println("No login information for " + args[1])
			}
		} else {
			fmt.Println("Usage: `get account.com`")
		}
		return nil

	case "regen":
		if len(args) == 2 {
			if prog.loginsLut[args[1]] != nil {
				prog.loginsLut[args[1]].MakeNewRandomSeed()
				prog.write()
				fmt.Println("Password reset for " + args[1])
			} else {
				fmt.Println("No login information for " + args[1])
			}
		} else {
			fmt.Println("Usage: `regen account.com`")
		}
		return nil
	case "edit":
		if len(args) == 2 {
			if prog.loginsLut[args[1]] != nil {
				prog.newLoginChildProgram.EditLogin(prog.loginsLut[args[1]])
				prog.setState(stateEdit)
			} else {
				fmt.Println("No login information for " + args[1])
			}
		} else {
			fmt.Println("Usage: `edit account.com`")
		}
		return nil
	case "list":
		prog.list(args)
		return nil
	case "exit":
		return errors.New("Exiting PORDS")
	default:
		fmt.Println(input + " is not a command. Enter 'help' to see command options.")
		return nil
	}
}

func (prog *GnocchiProgram) list(args []string) {
	l := len(prog.logins)

	if l == 0 {
		fmt.Println("No saved accounts.")
	} else {
		for i := 0; i < l; i++ {
			if i == 0 {
				fmt.Println("==================================")
			}
			prog.logins[i].PrintInfo(true)
			fmt.Println("==================================")
		}
	}
}

func (prog *GnocchiProgram) addLoginInfo(li *LoginInfo) {
	prog.logins = append(prog.logins, li)
	prog.loginsLut[li.siteName] = li
}

func (prog *GnocchiProgram) MasterPwIsSet() bool {
	return len(prog.MasterPw) > 0
}

func (prog *GnocchiProgram) init(cacheDir string) error {

	prog.cacheFilePath = cacheDir

	prog.logins = []*LoginInfo{}

	entryCursor := 0

	liBuilder := &LoginInfoBuilder{}

	if prog.Debug == true {
		liBuilder.prog = prog
	}

	var bizyte byte

	if prog.Debug == true {
		fmt.Println("Init GnocchiProgram")
	}

	_, err := os.Stat(prog.cacheFilePath)
	if err == nil {
		f, err := os.Open(prog.cacheFilePath)
		defer f.Close()
		if err != nil {
			return err
		}

		if prog.Debug == true {
			fmt.Println("Reading saved login info")
		}

		limit := 1024
		buf := make([]byte, limit)

		ct := 0
		for {
			n, err := f.Read(buf)

			if err == io.EOF {
				break
			}

			i := 0
			ct = 0
			for i < n {
				bizyte = buf[i]
				ct++
				if ct > limit {
					break
				}

				if bizyte == StartOfFile {
					entryCursor = 0
					liBuilder.flush()
					i += 3
				} else if bizyte == EndOfFile {
					prog.addLoginInfo(liBuilder.finish())
					break
				} else if bizyte == StartOfRecord && buf[i+1] == StartOfEntry {
					prog.addLoginInfo(liBuilder.finish())
					liBuilder.flush()
					entryCursor = 0
					i += 2
				} else if bizyte == StartOfEntry && entryCursor < 4 {
					entryCursor++
					i++
				} else {
					if entryCursor == 0 {
						liBuilder.appendSiteName(bizyte)
						i++
					} else if entryCursor == 1 {
						liBuilder.appendAccountName(bizyte)
						i++
					} else if entryCursor == 2 {
						liBuilder.appendSeed(bizyte)
						i++
					} else if entryCursor == 3 {
						liBuilder.setPwLength(bizyte)
						i++
					} else if entryCursor == 4 {
						liBuilder.appendSpecialChars(bizyte)
						i++
					}
				}
			}
		}
		return nil

	} else if os.IsNotExist(err) {
		if prog.Debug {
			fmt.Println(prog.cacheFilePath + " Does not exist")
		}
		return nil
	} else {
		return err
	}
}

func (prog *GnocchiProgram) write() error {

	if prog.Debug == true {
		fmt.Println("Attempting write to " + prog.cacheFilePath)
	}

	// TODO: some other file permission here than 0777?
	f, err := os.OpenFile(prog.cacheFilePath, os.O_RDWR|os.O_CREATE, 0777)
	defer f.Close()
	if err == nil {
		if prog.Debug == true {
			fmt.Println("Starting write of all login infos")
		}

		f.Write([]byte{StartOfFile})

		buf := make([]byte, 256)
		var liwr io.Reader

		l := len(prog.logins)
		for i := 0; i < l; i++ {
			liwr = prog.logins[i].GetReader()

			for {
				n, err := liwr.Read(buf)
				_, wErr := f.Write(buf[:n])
				if prog.Debug == true {
					fmt.Println("Writing account for " + prog.logins[i].siteName)
					fmt.Println("---------------------------------")
				}

				if err == io.EOF {
					if prog.Debug == true {
						fmt.Println("Finished write on LoginInfo")
						fmt.Println("Site Name: " + prog.logins[i].siteName)
						fmt.Println("---------------------------------")
					}
					break
				} else if err != nil {
					if prog.Debug == true {
						fmt.Println("Unknown error on LoginInfoReader")
					}
					break
				}

				if wErr != nil {
					if prog.Debug == true {
						fmt.Println("File write error!")
						fmt.Println(err.Error())
					}
					return wErr
				}

			}
		}
		f.Write([]byte{EndOfFile})
		return nil
	}
	fmt.Println("Error opening cache file!")
	fmt.Println(err.Error())
	return err
}

// StartProgram - Initializes new GnocchiProgram
func StartProgram(debug bool) (*GnocchiProgram, error) {
	prog := &GnocchiProgram{
		state:     stateMain,
		loginsLut: make(map[string]*LoginInfo),
	}

	if debug == true {
		prog.Debug = debug
	}

	prog.newLoginChildProgram = &NewLoginProgramModule{}

	homedir, err := homedir.Dir()
	fmt.Println(homedir)

	if err != nil {
		return prog, err
	}

	err = prog.init(homedir + "/.pords_cache")
	if err != nil {
		fmt.Println("Problem initializing Pords")
		return prog, err
	}

	return prog, nil
}
