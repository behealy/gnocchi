package program

import (
	"fmt"
	"strconv"
)

type phase int

const (
	newLoginStateRecordSitename phase = iota
	newLoginStateRecordAccountName
	newLoginStateRecordSpecialChars
	newLoginStateRecordPWLength
	newLoginStateConfirm
)

type NewLoginProgramModule struct {
	state        phase
	writtenLinfo *LoginInfo
	ProgramModule
}

func (prog *NewLoginProgramModule) NewLogin(wrLinfo *LoginInfo) {
	prog.state = newLoginStateRecordSitename
	prog.writtenLinfo = wrLinfo
}

func (prog *NewLoginProgramModule) EditLogin(wrLinfo *LoginInfo) {
	prog.state = newLoginStateRecordSpecialChars
	prog.writtenLinfo = wrLinfo
}

func (prog *NewLoginProgramModule) PrintPrompt() {
	switch prog.state {
	case newLoginStateRecordSitename:
		fmt.Println("Enter site name: ")
	case newLoginStateRecordAccountName:
		fmt.Println("Enter username for account: ")
	case newLoginStateRecordSpecialChars:
		specCharsMsg := "using no special chars"
		if len(prog.writtenLinfo.specialChars) > 0 {
			specCharsMsg = "using \"" + prog.writtenLinfo.specialChars + "\""
		}
		fmt.Println("Enter special characters to use in this password (currently " + specCharsMsg + ", to leave unchanged, just press enter): ")
	case newLoginStateRecordPWLength:
		fmt.Println("Enter desired password length (currently set to " + strconv.Itoa(int(prog.writtenLinfo.generatedPwLength)) + "): ")
	case newLoginStateConfirm:
		fmt.Println("Is the account info you entered correct? (confirm yes/no)")
		prog.writtenLinfo.PrintInfo(true)
	}
}

func (prog *NewLoginProgramModule) HandleInput(input string) error {

	if input == "cancel" {
		return cancelErr
	}

	if prog.state == newLoginStateRecordSitename {
		if prog.writtenLinfo != nil {
			prog.writtenLinfo.siteName = input
			prog.state = newLoginStateRecordAccountName
		} else {
			panic("No login info to write to!")
		}

	} else if prog.state == newLoginStateRecordAccountName {
		if prog.writtenLinfo != nil {
			prog.writtenLinfo.userName = input
			prog.state = newLoginStateRecordSpecialChars
		} else {
			panic("No login info to write to!")
		}
	} else if prog.state == newLoginStateRecordSpecialChars {
		err := prog.writtenLinfo.SetAllowedSpecialChars(input)
		if err != nil {
			fmt.Println(err)
		} else {
			prog.state = newLoginStateRecordPWLength
		}
	} else if prog.state == newLoginStateRecordPWLength {
		if len(input) == 0 {
			prog.state = newLoginStateConfirm
		}

		if length, serr := strconv.Atoi(input); serr == nil {
			err := prog.writtenLinfo.SetGeneratedPwLength(length)
			if err != nil {
				fmt.Println(err)
			} else {
				prog.state = newLoginStateConfirm
			}
		} else {
			fmt.Println("Couldn't set password length, use a number!")
		}
	} else if prog.state == newLoginStateConfirm {
		if input == "yes" {
			return finishedErr
		} else if input == "no" {
			return cancelErr
		}
	}
	return nil
}
