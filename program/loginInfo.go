package pords

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"time"
)

const defaultEncodingAlphabet string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
const defaultGeneratedPasswordLen int = 12
const randomSeedLength int = 8

func randomInt(min, max int) int {
	return min + rand.Intn(max-min)
}

func randomSeed(len int) []byte {
	rand.Seed(time.Now().UnixNano())
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(randomInt(33, 126))
	}
	return bytes
}

type LoginInfoBuilder struct {
	siteName          []byte
	userName          []byte
	seed              []byte
	prog              *PordsProgram
	generatedPwLength int
	specialChars      []byte
}

func (libr *LoginInfoBuilder) finish() *LoginInfo {
	linfo := &LoginInfo{
		siteName:          string(libr.siteName),
		userName:          string(libr.userName),
		encoder:           base64.StdEncoding,
		seed:              libr.seed,
		generatedPwLength: libr.generatedPwLength,
		specialChars:      string(libr.specialChars),
	}
	linfo.SetAllowedSpecialChars(linfo.specialChars)
	return linfo
}

func (libr *LoginInfoBuilder) flush() {
	libr.siteName = []byte{}
	libr.userName = []byte{}
	libr.seed = []byte{}
	libr.generatedPwLength = 0
	libr.specialChars = []byte{}
}

func (libr *LoginInfoBuilder) appendSiteName(char byte) {
	libr.siteName = append(libr.siteName, char)
	if libr.prog != nil && libr.prog.Debug == true {
		fmt.Println("siteName: " + string(libr.siteName))
	}

}

func (libr *LoginInfoBuilder) appendAccountName(char byte) {
	libr.userName = append(libr.userName, char)
	if libr.prog != nil && libr.prog.Debug == true {
		fmt.Println("userName: " + string(libr.userName))
	}
}

func (libr *LoginInfoBuilder) appendSeed(char byte) {
	libr.seed = append(libr.seed, char)
	if libr.prog != nil && libr.prog.Debug == true {
		fmt.Println("siteName: " + string(libr.siteName))
	}
}

func (libr *LoginInfoBuilder) setPwLength(length byte) {
	libr.generatedPwLength = int(length)
	if libr.prog != nil && libr.prog.Debug == true {
		fmt.Println("generatedPwLength: " + string(length))
		fmt.Println(length)
	}
}

func (libr *LoginInfoBuilder) appendSpecialChars(char byte) {
	libr.specialChars = append(libr.specialChars, char)
	if libr.prog != nil && libr.prog.Debug == true {
		fmt.Println("specialChars: " + string(char))
		fmt.Println(char)
	}
}

type LoginInfo struct {
	siteName          string
	userName          string
	encoder           *base64.Encoding
	seed              []byte
	generatedPwLength int
	specialChars      string
}

func NewBlankLoginInfo() LoginInfo {
	return LoginInfo{
		encoder:           base64.StdEncoding,
		seed:              randomSeed(randomSeedLength),
		generatedPwLength: defaultGeneratedPasswordLen,
		specialChars:      "",
	}
}

func (linfo *LoginInfo) MakeNewRandomSeed() {
	linfo.seed = randomSeed(randomSeedLength)
}

func (linfo *LoginInfo) SetGeneratedPwLength(len int) error {
	if len > 32 {
		return errors.New("Cannot make a password longer than 32 characters")
	} else {
		linfo.generatedPwLength = len
		return nil
	}
}

func (linfo *LoginInfo) SetAllowedSpecialChars(chars string) error {
	if len(chars) > 0 {
		maxAllowedSpecialChars := linfo.generatedPwLength / 2

		if len(chars) > maxAllowedSpecialChars {
			return errors.New("You can only set up to " + string(maxAllowedSpecialChars) + " special characters.")
		}
		linfo.specialChars = chars
	}

	return nil
}

func (linfo *LoginInfo) GetAcctPassword(masterPw []byte) string {
	concatPassword := []byte{}
	concatPassword = append(concatPassword, masterPw...)
	concatPassword = append(concatPassword, []byte(linfo.siteName+linfo.userName)...)
	concatPassword = append(concatPassword, linfo.seed...)

	hashed := sha256.Sum256(concatPassword)

	passwordBuf := make([]byte, linfo.encoder.EncodedLen(len(hashed)))
	linfo.encoder.Encode(passwordBuf, hashed[:])

	if len(linfo.specialChars) > 0 {
		linfo.swapInSpecialChars(passwordBuf)
	}

	return string(passwordBuf[:linfo.generatedPwLength])
}

func (linfo *LoginInfo) swapInSpecialChars(password []byte) {

	rand.Seed(int64(binary.BigEndian.Uint64(linfo.seed)))

	l := linfo.generatedPwLength
	swapPositions := make([]int, l)

	for i := 0; i < l; i++ {
		swapPositions[i] = i
	}
	rand.Shuffle(l, func(i, j int) {
		swapPositions[i], swapPositions[j] = swapPositions[j], swapPositions[i]
	})

	scLen := len(linfo.specialChars)
	for i := 0; i < scLen; i++ {
		password[swapPositions[i]] = byte(linfo.specialChars[i])
	}
}

func (linfo *LoginInfo) PrintInfo(settings bool) {
	fmt.Println("Site name: " + linfo.siteName)
	fmt.Println("User name: " + linfo.userName)
	if settings == true {
		fmt.Println("Settings -------------------------")
		fmt.Println("Special characters: " + linfo.specialChars)
		fmt.Println("Password Length: " + strconv.Itoa(int(linfo.generatedPwLength)))
	}
}

func (linfo *LoginInfo) setEncodingAlphabet(alphabet string) {
	if len(alphabet) != 64 {
		panic("encoding alphabet is not 64-bytes long")
	}
	linfo.encoder = base64.NewEncoding(alphabet)
}

func (linfo *LoginInfo) GetReader() io.Reader {
	written := []byte{StartOfRecord, StartOfEntry}

	written = append(written, linfo.siteName...)
	written = append(written, StartOfEntry)
	written = append(written, linfo.userName...)
	written = append(written, StartOfEntry)
	written = append(written, linfo.seed...)
	written = append(written, StartOfEntry)
	written = append(written, byte(uint8(linfo.generatedPwLength)))
	written = append(written, StartOfEntry)
	written = append(written, linfo.specialChars...)

	// fmt.Println(written, len(written))

	return LoginInfoReader{
		cursor:   0,
		written:  written,
		capacity: len(written),
	}
}

type LoginInfoReader struct {
	cursor   int
	written  []byte
	capacity int
}

func (linfR LoginInfoReader) Read(p []byte) (n int, err error) {

	l := len(p)

	err = nil
	n = 0

	for i := 0; i < l; i++ {
		if linfR.cursor == linfR.capacity {
			err = io.EOF
			break
		}
		p[i] = linfR.written[linfR.cursor]
		linfR.cursor++
		n = i + 1
	}
	return
}
